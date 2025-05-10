package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cespare/xxhash/v2"
)

type DirInfo struct {
	files map[string]struct{}
	dirs  map[string]DirInfo
}

func main() {
	start := time.Now()
	src, dst := parseFlags()
	src, err := filepath.Abs(src)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v\n", err)
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v\n", err)
	}
	srcDirInfo, err := getDirInfo(src)
	if err != nil {
		log.Fatalf("Error reading directory: %v\n", err)
	}
	dstDirInfo, err := getDirInfo(dst)
	if err != nil {
		log.Fatalf("Error reading directory: %v\n", err)
	}
	syncDirs(src, dst, srcDirInfo, dstDirInfo)
	fmt.Printf("Time taken: %s", time.Since(start))
}

// Parse flags required to run the program.
func parseFlags() (string, string) {
	srcPtr := flag.String("from", "./test-1", "Directory with files to sync from")
	flag.StringVar(srcPtr, "f", *srcPtr, "Alias for src")
	dstPtr := flag.String("to", "./test-2", "Directory with files to sync to")
	flag.StringVar(dstPtr, "t", *dstPtr, "Alias for dst")
	flag.Parse()
	return *srcPtr, *dstPtr
}

// Map directory info
func getDirInfo(absDirPath string) (DirInfo, error) {
	dirInfo := DirInfo{}
	items, err := os.ReadDir(absDirPath)
	if err != nil {
		return dirInfo, err
	}
	dirInfo.files = make(map[string]struct{})
	dirInfo.dirs = make(map[string]DirInfo)
	for _, item := range items {
		if item.IsDir() {
			dirName := item.Name()
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
		dirInfo.files[fileName] = struct{}{}
	}
	return dirInfo, nil
}

func syncDirs(srcPath, dstPath string, srcDirInfo, dstDirInfo DirInfo) {
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
			dstSubDirInfo = DirInfo{
				files: make(map[string]struct{}),
				dirs:  make(map[string]DirInfo),
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
