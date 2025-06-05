package lsync

import (
	"io"
	"log"
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

func PreviewSync(src, dst dirfetch.DirTree, srcPath, dstPath string) SyncPreview {
	dirSyncStruct := SyncPreview{
		Status:  StatusNone,
		Subdirs: make(map[string]SyncPreview),
		Files:   make(map[string]SyncStatus),
	}

	modified := false
	for fileName := range src.Files {
		srcFilePath, dstFilePath := filepath.Join(srcPath, fileName), filepath.Join(dstPath, fileName)
		dirSyncStruct.Files[fileName] = StatusNone
		_, ok := dst.Files[fileName]
		if !ok {
			dirSyncStruct.Files[fileName] = StatusCreated
			modified = true
			continue
		}
		ok, err := isSameFileContent(srcFilePath, dstFilePath)
		if err != nil {
			log.Println(err)
			continue
		}
		if ok {
			continue
		}
		delete(dst.Files, fileName)
	}

	for fileName := range dst.Files {
		dirSyncStruct.Files[fileName] = StatusDeleted
		modified = true
	}

	for subdirName, srcSubdirStruct := range src.Subdirs {
		srcSubdirPath, dstSubdirPath := filepath.Join(srcPath, subdirName), filepath.Join(dstPath, subdirName)
		dstSubdirStruct, ok := dst.Subdirs[subdirName]
		if !ok {
			empty := dirfetch.MakeEmptyDirTree()
			subdirSyncStruct := PreviewSync(srcSubdirStruct, empty, srcSubdirPath, dstSubdirPath)
			subdirSyncStruct.Status = StatusCreated
			dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
			modified = true
			continue
		}
		subdirSyncStruct := PreviewSync(srcSubdirStruct, dstSubdirStruct, srcSubdirPath, dstSubdirPath)
		dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
		if !modified && subdirSyncStruct.Status != StatusNone {
			modified = true
		}
		delete(dst.Subdirs, subdirName)
	}

	for subdirName := range dst.Subdirs {
		srcSubdirPath, dstSubdirPath := filepath.Join(srcPath, subdirName), filepath.Join(dstPath, subdirName)
		empty := dirfetch.MakeEmptyDirTree()
		subdirSyncStruct := PreviewSync(empty, dst.Subdirs[subdirName], srcSubdirPath, dstSubdirPath)
		subdirSyncStruct.Status = StatusDeleted
		dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
		modified = true
	}

	if modified {
		dirSyncStruct.Status = StatusModified
	}
	return dirSyncStruct
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
