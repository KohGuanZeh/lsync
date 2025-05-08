package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"path/filepath"
)

type DirInfo struct {
	files map[string]string
	dirs  map[string]DirInfo
}

func main() {
	src, dst := parseFlags()
	src, err := filepath.Abs(src)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v\n", err)
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v\n", err)
	}
	sha := sha1.New()
	srcDirInfo, err := getDirInfo(src, sha)
	if err != nil {
		log.Fatalf("Error reading directory: %v\n", err)
	}
	dstDirInfo, err := getDirInfo(dst, sha)
	if err != nil {
		log.Fatalf("Error reading directory: %v\n", err)
	}
	syncDirs(src, dst, srcDirInfo, dstDirInfo)
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
func getDirInfo(absDirPath string, sha hash.Hash) (DirInfo, error) {
	dirInfo := DirInfo{}
	items, err := os.ReadDir(absDirPath)
	if err != nil {
		return dirInfo, err
	}
	dirInfo.files = make(map[string]string)
	dirInfo.dirs = make(map[string]DirInfo)
	for _, item := range items {
		if item.IsDir() {
			dirName := item.Name()
			subdirInfo, err := getDirInfo(filepath.Join(absDirPath, dirName), sha)
			if err != nil {
				fmt.Printf("Error reading directory: %v\n", err)
				fmt.Println("Skipping...")
				continue
			}
			dirInfo.dirs[dirName] = subdirInfo
			continue
		}
		fileName := item.Name()
		data, err := os.ReadFile(filepath.Join(absDirPath, fileName))
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			fmt.Println("Skipping...")
			continue
		}
		sha.Write(data)
		hexDigest := hex.EncodeToString(sha.Sum(nil)[:8])
		sha.Reset()
		dirInfo.files[fileName] = hexDigest
	}
	return dirInfo, nil
}

func syncDirs(srcPath, dstPath string, srcDirInfo, dstDirInfo DirInfo) {
	for fileName, srcHash := range srcDirInfo.files {
		dstHash, ok := dstDirInfo.files[fileName]
		if !ok || srcHash != dstHash {
			srcFilePath := filepath.Join(srcPath, fileName)
			dstFilePath := filepath.Join(dstPath, fileName)
			copyFile(srcFilePath, dstFilePath)
		}
		delete(dstDirInfo.files, fileName)
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
		if !ok {
			err := os.Mkdir(dstSubDirPath, 0777)
			if err != nil {
				fmt.Printf("Cannot create subdirectory: %v\n", err)
			}
			dstSubDirInfo = DirInfo{
				files: make(map[string]string),
				dirs:  make(map[string]DirInfo),
			}
		}
		syncDirs(srcSubDirPath, dstSubDirPath, srcSubDirInfo, dstSubDirInfo)
		delete(dstDirInfo.dirs, dirName)
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
}
