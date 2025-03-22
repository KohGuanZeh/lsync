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
	"time"
)

func main() {
	dirPath := parseFlags()
	absDirPath, err := filepath.Abs(dirPath)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v\n", err)
	}
	start := time.Now()
	sha := sha1.New()
	dirWalk(absDirPath, sha, 255)
	fmt.Printf("Time taken: %s", time.Since(start))
}

// Parse flags required to run the program.
func parseFlags() string {
	dirPathPtr := flag.String("dir", "", "Directory for folder scan.")
	flag.StringVar(dirPathPtr, "d", *dirPathPtr, "Alias for dir.")
	flag.Parse()
	return *dirPathPtr
}

// Walk through items in given directory, printing filenames and their SHA1 values.
func dirWalk(absDirPath string, sha hash.Hash, maxDepth uint8) {
	items, err := os.ReadDir(absDirPath)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}
	fmt.Println()
	for _, item := range items {
		if item.IsDir() {
			dirname := item.Name()
			if maxDepth == 0 {
				fmt.Printf("Encountered directory at max depth: %s\n\n", dirname)
				continue
			}
			dirWalk(filepath.Join(absDirPath, dirname), sha, maxDepth-1)
			continue
		}
		filename := item.Name()
		fmt.Printf("Found file: %s\n", filename)
		data, err := os.ReadFile(filepath.Join(absDirPath, filename))
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}
		sha.Write(data)
		hexDigest := hex.EncodeToString(sha.Sum(nil)[:8])
		sha.Reset()
		// Only show the first 8 bytes of the SHA digest.
		fmt.Printf("File Contents SHA-1: %s\n\n", hexDigest)
	}
}
