package dirmap

import (
	"log"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
)

type DirStruct struct {
	Files   map[string]FileMetadata
	Subdirs map[string]DirStruct
}

type FileMetadata struct {
	ContentHash uint64
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
