package dirfetch

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/cespare/xxhash/v2"
)

type DirStruct struct {
	Files   map[string]FileMetadata
	Subdirs map[string]DirStruct
}

type NewDirStruct struct {
	Files   map[string]FileMetadata
	Subdirs map[string]*NewDirStruct
}

type DirStructResult struct {
	DirName   string
	DirStruct DirStruct
	Err       error
}

type FileMetadata struct {
	ContentHash uint64
}

type FileMetadataResult struct {
	FileName     string
	FileMetadata FileMetadata
	Err          error
}

type DirToProcess struct {
	path      string
	structPtr *NewDirStruct
}

type FileWorkerInfo struct {
	fileName  string
	filePath  string
	dirStruct *NewDirStruct
	wg        *sync.WaitGroup
}

func FetchDir(dirPath string) NewDirStruct {
	dirStruct := NewDirStruct{}
	mutex := new(sync.Mutex)
	fileWorkerPool := make(chan FileWorkerInfo, 50)
	for range 5 {
		go fileWorker(fileWorkerPool, mutex)
	}
	fileWorkerWg := new(sync.WaitGroup)
	dirsToProcess := []DirToProcess{{path: dirPath, structPtr: &dirStruct}}
	for len(dirsToProcess) > 0 {
		dir := dirsToProcess[0]
		dirsToProcess = dirsToProcess[1:]
		dirItems, err := os.ReadDir(dir.path)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, dirItem := range dirItems {
			if dirItem.IsDir() {
				subdirName := dirItem.Name()
				subdirPath := filepath.Join(dir.path, subdirName)
				subdirStruct := NewDirStruct{}
				mutex.Lock()
				dir.structPtr.Subdirs[subdirName] = &subdirStruct
				mutex.Unlock()
				dirsToProcess = append(dirsToProcess, DirToProcess{path: subdirPath, structPtr: &subdirStruct})
				continue
			}
			fileName := dirItem.Name()
			fileWorkerInfo := FileWorkerInfo{
				fileName:  fileName,
				filePath:  filepath.Join(dir.path, fileName),
				dirStruct: dir.structPtr,
				wg:        fileWorkerWg,
			}
			fileWorkerWg.Add(1)
			fileWorkerPool <- fileWorkerInfo
		}
	}
	fileWorkerWg.Wait()
	return dirStruct
}

func fileWorker(pool chan FileWorkerInfo, mutex *sync.Mutex) {
	for workerInfo := range pool {
		digest, err := hashFileContent(workerInfo.filePath)
		if err != nil {
			log.Println(err)
			workerInfo.wg.Done()
			continue
		}
		mutex.Lock()
		workerInfo.dirStruct.Files[workerInfo.fileName] = FileMetadata{ContentHash: digest}
		mutex.Unlock()
		workerInfo.wg.Done()
	}
}

func GetDirStruct(dirPath string) (DirStruct, error) {
	dirStruct := DirStruct{}
	dirItems, err := os.ReadDir(dirPath)
	if err != nil {
		return dirStruct, err
	}
	dirStruct.Files = make(map[string]FileMetadata)
	dirStruct.Subdirs = make(map[string]DirStruct)
	for _, dirItem := range dirItems {
		if dirItem.IsDir() {
			subdirName := dirItem.Name()
			subdirPath := filepath.Join(dirPath, dirItem.Name())
			subdirStruct, err := GetDirStruct(subdirPath)
			if err != nil {
				log.Println(err)
				continue
			}
			dirStruct.Subdirs[subdirName] = subdirStruct
			continue
		}
		fileName := dirItem.Name()
		digest, err := hashFileContent(filepath.Join(dirPath, fileName))
		if err != nil {
			log.Println(err)
			continue
		}
		dirStruct.Files[fileName] = FileMetadata{ContentHash: digest}
	}
	return dirStruct, nil
}

func GetDirStructAsync(dirName string, dirPath string, ch chan DirStructResult, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	dirStruct := DirStruct{}
	dirItems, err := os.ReadDir(dirPath)
	if err != nil {
		ch <- DirStructResult{
			DirName:   dirName,
			DirStruct: dirStruct,
			Err:       err,
		}
		return
	}
	subdirCh := make(chan DirStructResult, 5)
	fileCh := make(chan FileMetadataResult, 5)
	dirItemsWg := new(sync.WaitGroup)
	for _, dirItem := range dirItems {
		dirItemsWg.Add(1)
		name := dirItem.Name()
		path := filepath.Join(dirPath, name)
		if dirItem.IsDir() {
			go GetDirStructAsync(name, path, subdirCh, dirItemsWg)
		} else {
			go GetFileMetadataAsync(name, path, fileCh, dirItemsWg)
		}
	}

	go closeChannelsOnWaitDone(fileCh, subdirCh, dirItemsWg)

	dirStruct.Files = make(map[string]FileMetadata)
	dirStruct.Subdirs = make(map[string]DirStruct)
	for subdirCh != nil || fileCh != nil {
		select {
		case subdirRes, ok := <-subdirCh:
			if !ok {
				subdirCh = nil
				continue
			}
			if subdirRes.Err != nil {
				log.Println(subdirRes.Err)
				continue
			}
			dirStruct.Subdirs[subdirRes.DirName] = subdirRes.DirStruct
		case fileRes, ok := <-fileCh:
			if !ok {
				fileCh = nil
				continue
			}
			if fileRes.Err != nil {
				log.Println(fileRes.Err)
				continue
			}
			dirStruct.Files[fileRes.FileName] = fileRes.FileMetadata
		}
	}

	ch <- DirStructResult{
		DirName:   dirName,
		DirStruct: dirStruct,
		Err:       nil,
	}
}

func GetFileMetadataAsync(fileName string, filePath string, ch chan FileMetadataResult, wg *sync.WaitGroup) {
	defer wg.Done()
	fileMetadata := FileMetadata{}
	digest, err := hashFileContent(filePath)
	if err == nil {
		fileMetadata.ContentHash = digest
	}
	ch <- FileMetadataResult{
		FileName:     fileName,
		FileMetadata: fileMetadata,
		Err:          err,
	}
}

func closeChannelsOnWaitDone(fileCh chan FileMetadataResult, subdirCh chan DirStructResult, wg *sync.WaitGroup) {
	wg.Wait()
	close(fileCh)
	close(subdirCh)
}

func MakeEmptyDirStruct() DirStruct {
	return DirStruct{
		Files:   make(map[string]FileMetadata),
		Subdirs: make(map[string]DirStruct),
	}
}

func hashFileContent(filePath string) (uint64, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	return xxhash.Sum64(fileData), nil
}
