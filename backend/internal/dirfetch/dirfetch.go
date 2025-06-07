package dirfetch

import (
	"io/fs"
	"path/filepath"
)

type DirFetchResult struct {
	Err     error
	ItemMap DirItemMap
}

type DirItemMap struct {
	BasePath string
	RelPaths map[string]map[string]struct{}
}

func FetchDirAsync(path string) chan DirFetchResult {
	ch := make(chan DirFetchResult)
	go func() {
		ch <- FetchDir(path)
	}()
	return ch
}

func FetchDir(path string) DirFetchResult {
	itemMap := DirItemMap{
		BasePath: path,
		RelPaths: make(map[string]map[string]struct{}),
	}
	err := filepath.WalkDir(path, walkDirFunc(path, &itemMap))
	return DirFetchResult{
		Err:     err,
		ItemMap: itemMap,
	}
}

func walkDirFunc(root string, fetchResult *DirItemMap) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			// Root tree has already been created
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			fetchResult.RelPaths[relPath] = make(map[string]struct{})
			return nil
		}
		dirPath := filepath.Dir(relPath)
		fileName := d.Name()
		fetchResult.RelPaths[dirPath][fileName] = struct{}{}
		return nil
	}
}
