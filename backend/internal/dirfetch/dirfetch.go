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
	Subdirs map[string]*DirStruct
}

type FileMetadata struct {
	ContentHash uint64
}

type FileTask struct {
	parent *DirStruct
	mutex  *sync.Mutex
	name   string
	path   string
	wg     *sync.WaitGroup
}

type DirQueueItem struct {
	dirStruct *DirStruct
	mutex     *sync.Mutex
	path      string
}

func FetchDir(dirPath string) DirStruct {
	fileTaskCh := make(chan FileTask, 100)
	fileTaskWg := new(sync.WaitGroup)
	for i := 0; i < 5; i++ {
		go fileTaskWorker(fileTaskCh)
	}

	dirStruct := MakeEmptyDirStruct()
	rootItem := DirQueueItem{
		dirStruct: &dirStruct,
		mutex:     new(sync.Mutex),
		path:      dirPath,
	}
	dirQueue := []DirQueueItem{rootItem}
	for len(dirQueue) > 0 {
		dqi := dirQueue[0]
		dirQueue = dirQueue[1:]
		dirItems, err := os.ReadDir(dqi.path)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, dirItem := range dirItems {
			name := dirItem.Name()
			path := filepath.Join(dqi.path, name)
			if dirItem.IsDir() {
				subdirStruct := MakeEmptyDirStruct()
				nextDqi := DirQueueItem{
					dirStruct: &subdirStruct,
					mutex:     new(sync.Mutex),
					path:      path,
				}
				dirQueue = append(dirQueue, nextDqi)
				dqi.mutex.Lock()
				dqi.dirStruct.Subdirs[name] = &subdirStruct
				dqi.mutex.Unlock()
				continue
			}
			fileTaskWg.Add(1)
			fileTaskCh <- FileTask{
				parent: dqi.dirStruct,
				mutex:  dqi.mutex,
				name:   name,
				path:   path,
				wg:     fileTaskWg,
			}
		}
	}
	fileTaskWg.Wait()
	return dirStruct
}

func MakeEmptyDirStruct() DirStruct {
	return DirStruct{
		Files:   make(map[string]FileMetadata),
		Subdirs: make(map[string]*DirStruct),
	}
}

func fileTaskWorker(fileTaskCh <-chan FileTask) {
	for ft := range fileTaskCh {
		digest, err := hashFileContent(ft.path)
		if err != nil {
			log.Println(err)
			ft.wg.Done()
			continue
		}
		fileMetadata := FileMetadata{ContentHash: digest}
		ft.mutex.Lock()
		ft.parent.Files[ft.name] = fileMetadata
		ft.mutex.Unlock()
		ft.wg.Done()
	}
}

func hashFileContent(filePath string) (uint64, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	return xxhash.Sum64(fileData), nil
}
