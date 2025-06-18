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

const (
	STATUS_NONE     uint8 = 0b0001
	STATUS_DELETED  uint8 = 0b0010
	STATUS_CREATED  uint8 = 0b0100
	STATUS_MODIFIED uint8 = 0b1000
)

type SyncPreview struct {
	Status  uint8
	Subdirs map[string]*SyncPreview
	Files   map[string]uint8
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
	status  uint8
	files   map[string]uint8
}

type FileCmpTask struct {
	fileName    string
	srcFilePath string
	dstFilePath string
	cmpResCh    chan FileCmpTaskResult
	wg          *sync.WaitGroup
}

type FileCmpTaskResult struct {
	fileName   string
	isSameFile bool
}

func PreviewSync(src, dst dirfetch.DirItemMap) SyncPreview {
	wg := new(sync.WaitGroup)
	subdirTaskCh := make(chan SubdirTask, 100)
	fileCmpTaskCh := make(chan FileCmpTask, 100)
	resCh := make(chan SubDirTaskResult, 100)
	for range 10 {
		go subdirWorker(subdirTaskCh, fileCmpTaskCh, resCh, wg)
	}
	for range 5 {
		go fileCmpWorker(fileCmpTaskCh)
	}
	wg.Add(1)
	go iterDirItemMap(src, dst, subdirTaskCh, wg)
	go closeChannelOnWaitDone(resCh, wg)
	syncPreview := SyncPreview{
		Status:  0,
		Subdirs: make(map[string]*SyncPreview),
	}
	pathSep := string(os.PathSeparator)
	for result := range resCh {
		preview := &syncPreview
		subdirs := strings.Split(result.relPath, pathSep)
		if result.relPath == "." {
			// Root directory
			subdirs = nil
		}
		for i := 0; i < len(subdirs); i++ {
			preview.Status = preview.Status | result.status
			subdir := subdirs[i]
			subdirPreview, ok := preview.Subdirs[subdir]
			if !ok {
				subdirPreview = &SyncPreview{
					Status:  result.status,
					Subdirs: make(map[string]*SyncPreview),
				}
				preview.Subdirs[subdir] = subdirPreview
			}
			preview = subdirPreview
		}
		preview.Status = preview.Status | result.status
		preview.Files = result.files
	}
	// Close channels for goroutines to exit
	close(subdirTaskCh)
	close(fileCmpTaskCh)
	return syncPreview
}

func SyncWithPreview(src, dst string, preview SyncPreview, ignoreDelete bool) error {
	switch preview.Status {
	case STATUS_NONE:
		return nil
	case STATUS_CREATED:
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
	case STATUS_MODIFIED:
		if ignoreDelete {
			return nil
		}
		err := os.RemoveAll(dst)
		return err
	default:
		// In all other cases, the folder is modified
		for file, fileStatus := range preview.Files {
			dstPath := filepath.Join(dst, file)
			switch fileStatus {
			case STATUS_CREATED, STATUS_MODIFIED:
				srcPath := filepath.Join(src, file)
				err := copyFile(srcPath, dstPath)
				if err != nil {
					return err
				}
			case STATUS_DELETED:
				if ignoreDelete {
					break
				}
				err := os.Remove(dstPath)
				if err != nil {
					return err
				}
			}
		}
	}

	// Only if folder is created or modified, sync on subfolders.
	for subdir, subdirPreview := range preview.Subdirs {
		srcPath := filepath.Join(src, subdir)
		dstPath := filepath.Join(dst, subdir)
		err := SyncWithPreview(srcPath, dstPath, *subdirPreview, ignoreDelete)
		if err != nil {
			return err
		}
	}
	return nil
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

func subdirWorker(subdirTaskCh chan SubdirTask, fileCmpTaskCh chan FileCmpTask, resCh chan SubDirTaskResult, wg *sync.WaitGroup) {
	for task := range subdirTaskCh {
		taskRes := SubDirTaskResult{relPath: task.relPath, status: 0, files: make(map[string]uint8)}
		if task.srcSubdirItems == nil {
			// Subdir does not exist in source
			taskRes.files = make(map[string]uint8, len(task.dstSubdirItems))
			for fileName := range task.dstSubdirItems {
				taskRes.files[fileName] = STATUS_DELETED
			}
			taskRes.status = STATUS_DELETED
		} else if task.dstSubdirItems == nil {
			// Subdir does not exist in destination
			taskRes.files = make(map[string]uint8, len(task.srcSubdirItems))
			for fileName := range task.srcSubdirItems {
				taskRes.files[fileName] = STATUS_CREATED
			}
			taskRes.status = STATUS_CREATED
		} else {
			cmpWg := new(sync.WaitGroup)
			cmpResCh := make(chan FileCmpTaskResult, len(task.srcSubdirItems))
			for fileName := range task.srcSubdirItems {
				_, ok := task.dstSubdirItems[fileName]
				delete(task.dstSubdirItems, fileName)
				if ok {
					srcFilePath := filepath.Join(task.srcBasePath, task.relPath, fileName)
					dstFilePath := filepath.Join(task.dstBasePath, task.relPath, fileName)
					isSameSize, err := isSameFileSize(srcFilePath, dstFilePath)
					if err != nil {
						log.Println(err)
						continue
					}
					if isSameSize {
						cmpWg.Add(1)
						fileCmpTaskCh <- FileCmpTask{
							fileName:    fileName,
							srcFilePath: srcFilePath,
							dstFilePath: dstFilePath,
							cmpResCh:    cmpResCh,
							wg:          cmpWg,
						}
						continue
					}
					taskRes.files[fileName] = STATUS_MODIFIED
					taskRes.status = taskRes.status | STATUS_MODIFIED
					continue
				}
				taskRes.files[fileName] = STATUS_CREATED
				taskRes.status = taskRes.status | STATUS_CREATED
			}

			if len(task.dstSubdirItems) > 0 {
				taskRes.status = taskRes.status | STATUS_DELETED
			}
			for fileName := range task.dstSubdirItems {
				taskRes.files[fileName] = STATUS_DELETED
			}

			go closeChannelOnWaitDone(cmpResCh, cmpWg)
			for cmpRes := range cmpResCh {
				if !cmpRes.isSameFile {
					taskRes.files[cmpRes.fileName] = STATUS_MODIFIED
					taskRes.status = taskRes.status | STATUS_MODIFIED
				} else {
					taskRes.files[cmpRes.fileName] = STATUS_NONE
					taskRes.status = taskRes.status | STATUS_NONE
				}
			}
		}
		resCh <- taskRes
		wg.Done()
	}
}

func fileCmpWorker(ch chan FileCmpTask) {
	for fileCmpTask := range ch {
		res := FileCmpTaskResult{
			fileName: fileCmpTask.fileName,
		}
		isSameFile, err := isSameFileContent(fileCmpTask.srcFilePath, fileCmpTask.dstFilePath)
		if err != nil {
			// On error, do not overwrite file (treat as same file)
			log.Println(err)
			res.isSameFile = true
		} else {
			res.isSameFile = isSameFile
		}
		fileCmpTask.cmpResCh <- res
		fileCmpTask.wg.Done()
	}
}

func closeChannelOnWaitDone[T any](ch chan T, wg *sync.WaitGroup) {
	wg.Wait()
	close(ch)
}

func isSameFileSize(src, dst string) (bool, error) {
	srcFileInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}

	dstFileInfo, err := os.Stat(dst)
	if err != nil {
		return false, err
	}

	return srcFileInfo.Size() == dstFileInfo.Size(), nil
}

func isSameFileContent(src, dst string) (bool, error) {
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
