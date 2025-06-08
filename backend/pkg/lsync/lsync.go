package lsync

import (
	"io"
	"lsync/backend/internal/dirfetch"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
)

type SyncStatus string

const (
	StatusNone     SyncStatus = "None"
	StatusCreated  SyncStatus = "Created"
	StatusModified SyncStatus = "Modified"
	StatusDeleted  SyncStatus = "Deleted"
)

type SyncPreview struct {
	Status  SyncStatus
	Subdirs map[string]SyncPreview
	Files   map[string]SyncStatus
}

type SubDirTask struct {
	srcBasePath    string
	dstBasePath    string
	relPath        string
	srcSubdirItems map[string]struct{}
	dstSubdirItems map[string]struct{}
}

func PreviewSync(src, dst dirfetch.DirItemMap) SyncPreview {
	subdirTaskCh := make(chan SubDirTask)
	for i := 0; i < 10; i++ {
		go subdirCmpWorker(subdirTaskCh)
	}

	for relPath, srcSubdirItem := range src.RelPaths {
		dstSubdirItem, ok := dst.RelPaths[relPath]
		if ok {
			subdirTask := SubDirTask{
				srcBasePath:    src.BasePath,
				dstBasePath:    dst.BasePath,
				relPath:        relPath,
				srcSubdirItems: srcSubdirItem,
				dstSubdirItems: dstSubdirItem,
			}
			subdirTaskCh <- subdirTask
		} else {
			// Add new entry to result...
		}
		delete(dst.RelPaths, relPath)
	}

	for relPath, dstSubdirItem := range dst.RelPaths {
		// Look at the remaining keys...
		// These are keys missing from source...
	}

	// Collect results in a map (same as prev)
	// Collapse results into a tree
	return SyncPreview{}
}

func subdirCmpWorker(subdirTaskCh chan SubDirTask) {
	for subdirTask := range subdirTaskCh {
		// subdir comparator
		// Create a worker for comparing files?
		// We can do it sequentially until we feel it is same.
		// Create a channel to get the result, and a waitgroup...
		// So also need a filecmp channel that is passed... so we can channel to it.
	}
}

func SyncWithPreview(src, dst string, preview SyncPreview, ignoreDelete bool) error {
	switch preview.Status {
	case StatusCreated:
		err := os.Mkdir(dst, 0755)
		if err != nil {
			return err
		}
		for file := range preview.Files {
			srcPath := filepath.Join(src, file)
			dstPath := filepath.Join(dst, file)
			err := copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	case StatusModified:
		for file, fileStatus := range preview.Files {
			dstPath := filepath.Join(dst, file)
			switch fileStatus {
			case StatusCreated, StatusModified:
				srcPath := filepath.Join(src, file)
				err := copyFile(srcPath, dstPath)
				if err != nil {
					return err
				}
			case StatusDeleted:
				if ignoreDelete {
					break
				}
				err := os.Remove(dstPath)
				if err != nil {
					return err
				}
			}
		}
	case StatusDeleted:
		if ignoreDelete {
			break
		}
		err := os.RemoveAll(dst)
		if err != nil {
			return err
		}
	}

	if preview.Status == StatusCreated || preview.Status == StatusModified {
		for subdir, subdirPreview := range preview.Subdirs {
			srcPath := filepath.Join(src, subdir)
			dstPath := filepath.Join(dst, subdir)
			err := SyncWithPreview(srcPath, dstPath, subdirPreview, ignoreDelete)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func isSameFileContent(src, dst string) (bool, error) {
	srcFileInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	dstFileInfo, err := os.Stat(dst)
	if err != nil {
		return false, err
	}

	if srcFileInfo.Size() != dstFileInfo.Size() {
		return false, nil
	}

	srcFile, err := os.ReadFile(src)
	if err != nil {
		return false, err
	}
	dstFile, err := os.ReadFile(dst)
	if err != nil {
		return false, err
	}

	srcDigest, dstDigest := xxhash.Sum64(srcFile), xxhash.Sum64(dstFile)
	return srcDigest == dstDigest, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}
