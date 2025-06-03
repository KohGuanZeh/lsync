package dirfetch

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
)

type DirTree struct {
	Subdirs map[string]*DirTree
	Files   map[string]struct{}
}

func FetchDirAsync(dirpath string) chan *DirTree {
	ch := make(chan *DirTree)
	go func() {
		ch <- FetchDir(dirpath)
	}()
	return ch
}

func FetchDir(dirpath string) *DirTree {
	rootTree := MakeEmptyDirTree()
	err := filepath.WalkDir(dirpath, walkDirFunc(dirpath, rootTree))
	if err != nil {
		log.Println(err)
		return nil
	}
	return rootTree
}

func walkDirFunc(root string, rootTree *DirTree) fs.WalkDirFunc {
	pathMap := make(map[string]*DirTree)
	pathMap[root] = rootTree
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			// Root tree has already been created
			return nil
		}

		parentDir := filepath.Dir(path)
		parentTree, ok := pathMap[parentDir]
		if !ok {
			return fmt.Errorf("cannot find parent DirTree for path: %s", parentDir)
		}

		name := d.Name()
		if d.IsDir() {
			subdirTree := MakeEmptyDirTree()
			parentTree.Subdirs[name] = subdirTree
			pathMap[path] = subdirTree
		} else {
			parentTree.Files[name] = struct{}{}
		}

		return nil
	}
}

func MakeEmptyDirTree() *DirTree {
	tree := DirTree{
		Subdirs: make(map[string]*DirTree),
		Files:   make(map[string]struct{}),
	}
	return &tree
}
