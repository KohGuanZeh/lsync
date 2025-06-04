package dirfetch

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type FetchDirResult struct {
	DirTree DirTree
	Err     error
}

type DirTree struct {
	Subdirs map[string]DirTree
	Files   map[string]struct{}
}

func FetchDirAsync(dirpath string) chan FetchDirResult {
	ch := make(chan FetchDirResult)
	go func() {
		ch <- FetchDir(dirpath)
	}()
	return ch
}

func FetchDir(dirpath string) FetchDirResult {
	tree := MakeEmptyDirTree()
	err := filepath.WalkDir(dirpath, walkDirFunc(dirpath, &tree))
	if err != nil {
		return FetchDirResult{
			DirTree: DirTree{},
			Err:     err,
		}
	}
	return FetchDirResult{
		DirTree: tree,
		Err:     nil,
	}
}

func walkDirFunc(rootPath string, rootTree *DirTree) fs.WalkDirFunc {
	pathSep := string(os.PathSeparator)
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == rootPath {
			// Root tree has already been created
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		parts := strings.Split(relPath, pathSep)

		tree := *rootTree
		for i := 0; i < len(parts)-1; i++ {
			subtree, ok := tree.Subdirs[parts[i]]
			if !ok {
				return fmt.Errorf("cannot find parent DirTree for path: %s", path)
			}
			tree = subtree
		}

		name := parts[len(parts)-1]
		if d.IsDir() {
			tree.Subdirs[name] = MakeEmptyDirTree()
		} else {
			tree.Files[name] = struct{}{}
		}
		return nil
	}
}

func MakeEmptyDirTree() DirTree {
	return DirTree{
		Subdirs: make(map[string]DirTree),
		Files:   make(map[string]struct{}),
	}
}
