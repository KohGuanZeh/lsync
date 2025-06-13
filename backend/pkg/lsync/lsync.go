package lsync

import (
	"io"
	"log"
	"lsync/backend/internal/dirfetch"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cespare/xxhash/v2"
)

type SyncStatus string

const (
	StatusNone     SyncStatus = "None"
	StatusCreated  SyncStatus = "Created"
	StatusDeleted  SyncStatus = "Deleted"
	StatusModified SyncStatus = "Modified"
)

type SyncPreview struct {
	Status  SyncStatus
	Subdirs map[string]*SyncPreview
	Files   map[string]SyncStatus
}

type SubdirTask struct {
	srcBasePath    string
	dstBasePath    string
	relPath        string
	srcSubdirItems map[string]struct{}
	dstSubdirItems map[string]struct{}
}

type SubDirTaskResult struct {
	relPath string
	status  SyncStatus
	files   map[string]SyncStatus
}

func PreviewSync(src, dst dirfetch.DirItemMap) SyncPreview {
	wg := new(sync.WaitGroup)
	taskCh := make(chan SubdirTask, 100)
	resCh := make(chan SubDirTaskResult, 100)
	for range 10 {
		go subdirWorker(taskCh, resCh, wg)
	}
	wg.Add(1)
	go iterDirItemMap(src, dst, taskCh, wg)
	go closeChannelOnWaitDone(resCh, wg)
	syncPreview := SyncPreview{
		Status:  StatusNone,
		Subdirs: make(map[string]*SyncPreview),
	}
	pathSep := string(os.PathSeparator)
	for result := range resCh {
		parent := &syncPreview
		// Root directory
		if result.relPath == "." {
			parent.Files = result.files
			continue
		}
		subdirs := strings.Split(result.relPath, pathSep)
		for i := 0; i < len(subdirs); i++ {
			subdir := subdirs[i]
			_, ok := parent.Subdirs[subdir]
			if !ok {
				subdirPreview := SyncPreview{
					Status:  StatusNone,
					Subdirs: make(map[string]*SyncPreview),
				}
				parent.Subdirs[subdir] = &subdirPreview
			}
			if i == len(subdirs)-1 {
				targetSubdir := parent.Subdirs[subdir]
				targetSubdir.Files = result.files
				if targetSubdir.Status != result.status {
					if len(targetSubdir.Subdirs) == 0 {
						targetSubdir.Status = result.status
					} else if targetSubdir.Status != result.status {
						targetSubdir.Status = StatusModified
					}
				}
			} else {
				parent = parent.Subdirs[subdir]
				if parent.Status != result.status {
					parent.Status = StatusModified
				}
			}
		}
		// Find a way to resolve sync status
	}
	// Close task channel for goroutines to exit.
	close(taskCh)
	return syncPreview
}

func iterDirItemMap(src, dst dirfetch.DirItemMap, ch chan SubdirTask, wg *sync.WaitGroup) {
	defer wg.Done()
	workerTask := SubdirTask{
		srcBasePath: src.BasePath,
		dstBasePath: dst.BasePath,
	}
	for relPath, srcSubdirItems := range src.RelPaths {
		dstSubdirItems, ok := dst.RelPaths[relPath]
		if !ok {
			dstSubdirItems = nil
		}
		workerTask.relPath = relPath
		workerTask.srcSubdirItems = srcSubdirItems
		workerTask.dstSubdirItems = dstSubdirItems
		wg.Add(1)
		ch <- workerTask
		delete(dst.RelPaths, relPath)
	}

	workerTask.srcSubdirItems = nil
	for relPath, dstSubdirItems := range dst.RelPaths {
		workerTask.relPath = relPath
		workerTask.dstSubdirItems = dstSubdirItems
		wg.Add(1)
		ch <- workerTask
	}
}

func subdirWorker(taskCh chan SubdirTask, resCh chan SubDirTaskResult, wg *sync.WaitGroup) {
	for task := range taskCh {
		taskRes := SubDirTaskResult{relPath: task.relPath, status: StatusNone, files: make(map[string]SyncStatus)}
		if task.srcSubdirItems == nil {
			// Subdir does not exist in source
			taskRes.files = make(map[string]SyncStatus, len(task.dstSubdirItems))
			for fileName := range task.dstSubdirItems {
				taskRes.files[fileName] = StatusDeleted
			}
			taskRes.status = StatusDeleted
		} else if task.dstSubdirItems == nil {
			// Subdir does not exist in destination
			taskRes.files = make(map[string]SyncStatus, len(task.srcSubdirItems))
			for fileName := range task.srcSubdirItems {
				taskRes.files[fileName] = StatusCreated
			}
			taskRes.status = StatusCreated
		} else {
			for fileName := range task.srcSubdirItems {
				_, ok := task.dstSubdirItems[fileName]
				if ok {
					// File comparison here
					srcFilePath := filepath.Join(task.srcBasePath, task.relPath, fileName)
					dstFilePath := filepath.Join(task.dstBasePath, task.relPath, fileName)
					// Perhaps introduce concurrency for file comparision
					// Can also look into improving speed for comparision via mmap etc.
					ok, err := isSameFileContent(srcFilePath, dstFilePath)
					if err != nil {
						log.Println(err)
						// Do not modify file content if there is an error.
						ok = true
					}
					if ok {
						taskRes.files[fileName] = StatusNone
					} else {
						taskRes.files[fileName] = StatusModified
						taskRes.status = StatusModified
					}
				} else {
					taskRes.files[fileName] = StatusCreated
					taskRes.status = StatusModified
				}
				delete(task.dstSubdirItems, fileName)
			}

			for fileName := range task.dstSubdirItems {
				taskRes.files[fileName] = StatusDeleted
				taskRes.status = StatusModified
			}
		}
		resCh <- taskRes
		wg.Done()
	}
}

func closeChannelOnWaitDone[T any](ch chan T, wg *sync.WaitGroup) {
	wg.Wait()
	close(ch)
}

func SyncWithPreview(src, dst string, preview SyncPreview, ignoreDelete bool) error {
	switch preview.Status {
	case StatusCreated:
		err := os.Mkdir(dst, 0755)
		if err != nil {
			return err
		}
		for file := range preview.Files {
			srcPath := filepath.Join(src, file)
			dstPath := filepath.Join(dst, file)
			err := copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	case StatusModified:
		for file, fileStatus := range preview.Files {
			dstPath := filepath.Join(dst, file)
			switch fileStatus {
			case StatusCreated, StatusModified:
				srcPath := filepath.Join(src, file)
				err := copyFile(srcPath, dstPath)
				if err != nil {
					return err
				}
			case StatusDeleted:
				if ignoreDelete {
					break
				}
				err := os.Remove(dstPath)
				if err != nil {
					return err
				}
			}
		}
	case StatusDeleted:
		if ignoreDelete {
			break
		}
		err := os.RemoveAll(dst)
		if err != nil {
			return err
		}
	}

	if preview.Status == StatusCreated || preview.Status == StatusModified {
		for subdir, subdirPreview := range preview.Subdirs {
			srcPath := filepath.Join(src, subdir)
			dstPath := filepath.Join(dst, subdir)
			err := SyncWithPreview(srcPath, dstPath, *subdirPreview, ignoreDelete)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func isSameFileContent(src, dst string) (bool, error) {
	srcFileInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	dstFileInfo, err := os.Stat(dst)
	if err != nil {
		return false, err
	}

	if srcFileInfo.Size() != dstFileInfo.Size() {
		return false, nil
	}

	srcFile, err := os.ReadFile(src)
	if err != nil {
		return false, err
	}
	dstFile, err := os.ReadFile(dst)
	if err != nil {
		return false, err
	}

	srcDigest, dstDigest := xxhash.Sum64(srcFile), xxhash.Sum64(dstFile)
	return srcDigest == dstDigest, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}
