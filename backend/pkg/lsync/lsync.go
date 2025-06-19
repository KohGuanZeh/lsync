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

type SyncOpType uint8

const (
	SYNC_OP_COPY SyncOpType = iota
	SYNC_OP_DELETE
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

type SyncOpTask struct {
	opType SyncOpType
	// For COPY, element 0 will be source.
	paths []string
}

type PreviewQueueItem struct {
	preview *SyncPreview
	srcPath string
	dstPath string
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

func SyncWithPreview(src, dst string, preview SyncPreview, ignoreDelete bool) {
	wg := new(sync.WaitGroup)
	taskCh := make(chan SyncOpTask, 100)
	for range 5 {
		wg.Add(1)
		go syncWorker(taskCh, wg)
	}
	previewQueue := []PreviewQueueItem{{preview: &preview, srcPath: src, dstPath: dst}}
	for len(previewQueue) > 0 {
		curr := previewQueue[0]
		previewQueue = previewQueue[1:]
		switch curr.preview.Status {
		case STATUS_NONE:
			continue
		case STATUS_DELETED:
			if ignoreDelete {
				continue
			}
			taskCh <- SyncOpTask{
				opType: SYNC_OP_DELETE,
				paths:  []string{curr.dstPath},
			}
			continue
		case STATUS_CREATED:
			err := os.Mkdir(curr.dstPath, 0755)
			if err != nil {
				log.Println(err)
				continue
			}
			for fileName := range curr.preview.Files {
				srcPath := filepath.Join(curr.srcPath, fileName)
				dstPath := filepath.Join(curr.dstPath, fileName)
				taskCh <- SyncOpTask{
					opType: SYNC_OP_COPY,
					paths:  []string{srcPath, dstPath},
				}
			}
		default:
			// In all other cases, the directory is modified
			for fileName, fileStatus := range curr.preview.Files {
				dstPath := filepath.Join(curr.dstPath, fileName)
				switch fileStatus {
				case STATUS_NONE:
					continue
				case STATUS_DELETED:
					if ignoreDelete {
						continue
					}
					taskCh <- SyncOpTask{
						opType: SYNC_OP_DELETE,
						paths:  []string{dstPath},
					}
				default:
					srcPath := filepath.Join(curr.srcPath, fileName)
					taskCh <- SyncOpTask{
						opType: SYNC_OP_COPY,
						paths:  []string{srcPath, dstPath},
					}
				}
			}
		}
		// Append remaining subdirectories to sync
		for subdirName, subdirPreview := range curr.preview.Subdirs {
			srcPath := filepath.Join(curr.srcPath, subdirName)
			dstPath := filepath.Join(curr.dstPath, subdirName)
			previewQueue = append(previewQueue, PreviewQueueItem{
				preview: subdirPreview,
				srcPath: srcPath,
				dstPath: dstPath,
			})
		}
	}
	close(taskCh)
	wg.Wait()
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

func syncWorker(ch chan SyncOpTask, wg *sync.WaitGroup) {
	for syncOpTask := range ch {
		switch syncOpTask.opType {
		case SYNC_OP_COPY:
			src := syncOpTask.paths[0]
			dsts := syncOpTask.paths[1:]
			copyFiles(src, dsts)
		case SYNC_OP_DELETE:
			for _, path := range syncOpTask.paths {
				err := os.RemoveAll(path)
				if err != nil {
					log.Printf("Failed to Sync for %v\n", path)
					log.Println(err)
				}
			}
		}
	}
	wg.Done()
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

func copyFiles(src string, dsts []string) {
	srcFile, err := os.Open(src)
	if err != nil {
		log.Printf("Failed to Open %v\n", src)
		log.Println(err)
		return
	}
	for _, dst := range dsts {
		dstFile, err := os.Create(dst)
		if err != nil {
			log.Println(err)
		}
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			log.Println(err)
		}
		dstFile.Close()
	}
	srcFile.Close()
}
