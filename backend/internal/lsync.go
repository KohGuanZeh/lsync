package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cespare/xxhash/v2"
)

type DirStruct struct {
	files map[string]FileSyncData
	dirs  map[string]DirStruct
}

type FileSyncData struct {
	hashExists   bool
	contentHash  uint64
	lastModified time.Time
} 

// Map directory info
func getDirInfo(absDirPath string) (DirStruct, error) {
	dirInfo := DirStruct{}
	items, err := os.ReadDir(absDirPath)
	if err != nil {
		return dirInfo, err
	}
	dirInfo.files = make(map[string]FileSyncData)
	dirInfo.dirs = make(map[string]DirStruct)
	for _, item := range items {
		if item.IsDir() {
			dirName := item.Name()
			if dirName == ".lsync" {
				continue
			}
			subdirInfo, err := getDirInfo(filepath.Join(absDirPath, dirName))
			if err != nil {
				fmt.Printf("Error reading directory: %v\n", err)
				fmt.Println("Skipping...")
				continue
			}
			dirInfo.dirs[dirName] = subdirInfo
			continue
		}
		fileName := item.Name()
		fileInfo, err := os.Stat(filepath.Join(absDirPath, fileName))
		if err != nil {
			fmt.Printf("Error reading file info: %v\n", err)
			fmt.Println("Skipping...")
			continue
		}
		dirInfo.files[fileName] = FileSyncData{hashExists: false, lastModified: fileInfo.ModTime()}
	}
	return dirInfo, nil
}

func syncDirs(srcPath, dstPath string, srcDirInfo, dstDirInfo DirStruct) {
	for fileName := range srcDirInfo.files {
		srcFilePath := filepath.Join(srcPath, fileName)
		dstFilePath := filepath.Join(dstPath, fileName)
		_, ok := dstDirInfo.files[fileName]
		delete(dstDirInfo.files, fileName)
		if !ok {
			copyFile(srcFilePath, dstFilePath)
			continue
		}
		isSameContent, err := isSameFileContent(srcFilePath, dstFilePath)
		if err != nil {
			fmt.Printf("Error checking file contents: %v\n", err)
		}
		if !isSameContent {
			copyFile(srcFilePath, dstFilePath)
		}
	}
	for fileName := range dstDirInfo.files {
		filePath := filepath.Join(dstPath, fileName)
		err := os.Remove(filePath)
		if err != nil {
			fmt.Printf("Cannot delete file: %v\n", err)
		}
	}

	for dirName, srcSubDirInfo := range srcDirInfo.dirs {
		srcSubDirPath := filepath.Join(srcPath, dirName)
		dstSubDirPath := filepath.Join(dstPath, dirName)
		dstSubDirInfo, ok := dstDirInfo.dirs[dirName]
		delete(dstDirInfo.dirs, dirName)
		if !ok {
			dirStat, err := os.Stat(srcSubDirPath)
			if err != nil {
				fmt.Printf("Cannot get source subdirectory info: %v\n", err)
				continue
			}
			perm := dirStat.Mode().Perm()
			err = os.Mkdir(dstSubDirPath, perm)
			if err != nil {
				fmt.Printf("Cannot create subdirectory: %v\n", err)
				continue
			}
			dstSubDirInfo = DirStruct{
				files: make(map[string]FileSyncData),
				dirs:  make(map[string]DirStruct),
			}
		}
		syncDirs(srcSubDirPath, dstSubDirPath, srcSubDirInfo, dstSubDirInfo)
	}
	for dirName := range dstDirInfo.dirs {
		dstSubDirPath := filepath.Join(dstPath, dirName)
		fmt.Println(dstSubDirPath)
		err := os.RemoveAll(dstSubDirPath)
		if err != nil {
			fmt.Printf("Cannot delete subdirectory: %v\n", err)
		}
	}
}

func isSameFileContent(srcFilePath, dstFilePath string) (bool, error) {
	srcFileInfo, err := os.Stat(srcFilePath)
	if err != nil {
		return false, err
	}
	dstFileInfo, err := os.Stat(dstFilePath)
	if err != nil {
		return false, err
	}
	if srcFileInfo.Size() != dstFileInfo.Size() {
		return false, nil
	}

	srcData, err := os.ReadFile(srcFilePath)
	if err != nil {
		return false, err
	}
	srcDigest := xxhash.Sum64(srcData)
	dstData, err := os.ReadFile(dstFilePath)
	if err != nil {
		return false, err
	}
	dstDigest := xxhash.Sum64(dstData)

	if srcDigest != dstDigest {
		return false, err
	}
	return true, nil
}

func copyFile(srcFilePath, dstFilePath string) {
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		fmt.Printf("Failed to open source file: %v\n", err)
		return
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dstFilePath)
	if err != nil {
		fmt.Printf("Failed to create destination file: %v\n", err)
		return
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		fmt.Printf("Failed to copy file: %v\n", err)
	}
	fmt.Println("FILE COPIED")
}