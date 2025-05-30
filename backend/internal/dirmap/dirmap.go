package dirmap

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
	defer wg.Done()
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
	subdirCh := make(chan DirStructResult)
	fileCh := make(chan FileMetadataResult)
	var dirWg sync.WaitGroup
	for _, dirItem := range dirItems {
		dirWg.Add(1)
		if dirItem.IsDir() {
			continue
		}
		fileName := dirItem.Name()
		filePath := filepath.Join(dirPath, fileName)
		go GetFileMetadataAsync(fileName, filePath, fileCh, &dirWg)
	}

	go closeChannelsOnWaitDone(fileCh, subdirCh, wg)

	dirStruct.Files = make(map[string]FileMetadata)
	dirStruct.Subdirs = make(map[string]DirStruct)
	for subdirRes := range subdirCh {
		if subdirRes.Err != nil {
			log.Println(subdirRes.Err)
			continue
		}
		dirStruct.Subdirs[subdirRes.DirName] = subdirRes.DirStruct
	}
	for fileRes := range fileCh {
		if fileRes.Err != nil {
			log.Println(fileRes.Err)
		}
		dirStruct.Files[fileRes.FileName] = fileRes.FileMetadata
	}

	ch <- DirStructResult{
		DirName:   dirName,
		DirStruct: dirStruct,
		Err:       nil,
	}
	close(ch)
}

func GetFileMetadataAsync(fileName string, filePath string, ch chan FileMetadataResult, wg *sync.WaitGroup) {
	defer wg.Done()
	fileMetadata := FileMetadata{}
	digest, err := hashFileContent(filePath)
	if err != nil {
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
