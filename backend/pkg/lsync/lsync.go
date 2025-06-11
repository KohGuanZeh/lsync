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
	StatusModified SyncStatus = "Modified"
	StatusDeleted  SyncStatus = "Deleted"
)

type SyncPreview struct {
	Status  SyncStatus
	Subdirs map[string]SyncPreview
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
		Subdirs: make(map[string]SyncPreview),
	}
	pathSep := string(os.PathSeparator)
	for result := range resCh {
		// Root directory
		if result.relPath == "." {
			syncPreview.Files = result.files
			continue
		}
		parentMap := syncPreview.Subdirs
		subdirs := strings.Split(result.relPath, pathSep)
		for i := 0; i < len(subdirs); i++ {
			subdir := subdirs[i]
			_, ok := parentMap[subdir]
			if !ok {
				parentMap[subdir] = SyncPreview{
					Status:  StatusNone,
					Subdirs: make(map[string]SyncPreview),
				}
			}
			if i == len(subdirs)-1 {
				// Not sure if there is a better way to resolve this...
				// Can use pointers but eventually need to dereference to pass back to front end.
				targetSyncPreview := parentMap[subdir]
				targetSyncPreview.Files = result.files
				parentMap[subdir] = targetSyncPreview
			} else {
				parentMap = parentMap[subdir].Subdirs
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
		taskRes := SubDirTaskResult{relPath: task.relPath, files: make(map[string]SyncStatus)}
		if task.srcSubdirItems == nil {
			// Subdir does not exist in source
			taskRes.files = make(map[string]SyncStatus, len(task.dstSubdirItems))
			for fileName := range task.dstSubdirItems {
				taskRes.files[fileName] = StatusDeleted
			}
		} else if task.dstSubdirItems == nil {
			// Subdir does not exist in destination
			taskRes.files = make(map[string]SyncStatus, len(task.srcSubdirItems))
			for fileName := range task.srcSubdirItems {
				taskRes.files[fileName] = StatusCreated
			}
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
					}
				} else {
					taskRes.files[fileName] = StatusCreated
				}
				delete(task.dstSubdirItems, fileName)
			}

			for fileName := range task.dstSubdirItems {
				taskRes.files[fileName] = StatusDeleted
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
			err := SyncWithPreview(srcPath, dstPath, subdirPreview, ignoreDelete)
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
