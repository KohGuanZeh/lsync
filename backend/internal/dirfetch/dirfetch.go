package dirfetch

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/cespare/xxhash/v2"
)

type DirTree struct {
	Files   map[string]FileMetadata
	Subdirs map[string]*DirTree
}

type FileMetadata struct {
	ContentHash uint64
}

type FileTask struct {
	parent *DirTree
	mutex  *sync.Mutex
	name   string
	path   string
	wg     *sync.WaitGroup
}

type DirQueueItem struct {
	dirTree *DirTree
	mutex   *sync.Mutex
	path    string
}

func FetchDirTreeAsync(dirPath string) chan *DirTree {
	ch := make(chan *DirTree)
	go func() {
		ch <- FetchDirTree(dirPath)
		close(ch)
	}()
	return ch
}

func FetchDirTree(dirPath string) *DirTree {
	fileTaskCh := make(chan FileTask, 100)
	fileTaskWg := new(sync.WaitGroup)
	for i := 0; i < 5; i++ {
		go fileTaskWorker(fileTaskCh)
	}

	dirTree := MakeEmptyDirTree()
	rootItem := DirQueueItem{
		dirTree: dirTree,
		mutex:   new(sync.Mutex),
		path:    dirPath,
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
				subdirTree := MakeEmptyDirTree()
				nextDqi := DirQueueItem{
					dirTree: subdirTree,
					mutex:   new(sync.Mutex),
					path:    path,
				}
				dirQueue = append(dirQueue, nextDqi)
				dqi.mutex.Lock()
				dqi.dirTree.Subdirs[name] = subdirTree
				dqi.mutex.Unlock()
				continue
			}
			fileTaskWg.Add(1)
			fileTaskCh <- FileTask{
				parent: dqi.dirTree,
				mutex:  dqi.mutex,
				name:   name,
				path:   path,
				wg:     fileTaskWg,
			}
		}
	}
	fileTaskWg.Wait()
	return dirTree
}

func MakeEmptyDirTree() *DirTree {
	emptyTree := DirTree{
		Files:   make(map[string]FileMetadata),
		Subdirs: make(map[string]*DirTree),
	}
	return &emptyTree
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
