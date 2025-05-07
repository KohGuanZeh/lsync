package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"log"
	"os"
	"path/filepath"
)

type DirInfo struct {
	files map[string]string
	dirs  map[string]DirInfo
}

func main() {
	from, to := parseFlags()
	from, err := filepath.Abs(from)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v\n", err)
	}
	to, err = filepath.Abs(to)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v\n", err)
	}
	sha := sha1.New()
	fromDirInfo, err := getDirInfo(from, sha)
	if err != nil {
		log.Fatalf("Error reading directory: %v\n", err)
	}
	toDirInfo, err := getDirInfo(to, sha)
	if err != nil {
		log.Fatalf("Error reading directory: %v\n", err)
	}
	syncDirs(from, to, fromDirInfo, toDirInfo)
}

// Parse flags required to run the program.
func parseFlags() (string, string) {
	fromPathPtr := flag.String("from", "./test-1", "Directory with files to sync from")
	flag.StringVar(fromPathPtr, "f", *fromPathPtr, "Alias for from")
	toPathPtr := flag.String("to", "./test-2", "Directory with files to sync to")
	flag.StringVar(toPathPtr, "t", *toPathPtr, "Alias for to")
	flag.Parse()
	return *fromPathPtr, *toPathPtr
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
			dirname := item.Name()
			subdirInfo, err := getDirInfo(filepath.Join(absDirPath, dirname), sha)
			if err != nil {
				fmt.Printf("Error reading directory: %v\n", err)
				fmt.Println("Skipping...")
				continue
			}
			dirInfo.dirs[dirname] = subdirInfo
			continue
		}
		filename := item.Name()
		data, err := os.ReadFile(filepath.Join(absDirPath, filename))
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			fmt.Println("Skipping...")
			continue
		}
		sha.Write(data)
		hexDigest := hex.EncodeToString(sha.Sum(nil)[:8])
		sha.Reset()
		dirInfo.files[filename] = hexDigest
	}
	return dirInfo, nil
}

func printDirInfo(dirInfo DirInfo) {
	if len(dirInfo.files) == 0 {
		fmt.Println("No files found")
	}
	fmt.Println("Files and Content Hash:")
	for k, v := range dirInfo.files {
		fmt.Printf("%s: %s\n", k, v)
	}
	if len(dirInfo.dirs) == 0 {
		fmt.Println("No Subdirectories found")
	}
	fmt.Println("Subdirectories:")
	for k, v := range dirInfo.dirs {
		fmt.Printf("%s:\n", k)
		printDirInfo(v)
	}
}

func syncDirs(from, to string, fromDirInfo, toDirInfo DirInfo) {
	// TO DO: Sync files from fromDir to toDir
	printDirInfo(fromDirInfo)
	fmt.Println()
	printDirInfo(toDirInfo)
}
