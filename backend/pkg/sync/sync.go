package sync

import (
	"io"
	"lsync/backend/internal/dirmap"
	"os"
	"path/filepath"
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

func PreviewSync(src, dst dirmap.DirStruct) SyncPreview {
	dirSyncStruct := SyncPreview{
		Status:  StatusNone,
		Subdirs: make(map[string]SyncPreview),
		Files:   make(map[string]SyncStatus),
	}

	modified := false
	for fileName, srcFileMetadata := range src.Files {
		dirSyncStruct.Files[fileName] = StatusNone
		dstFileMetadata, ok := dst.Files[fileName]
		if !ok {
			dirSyncStruct.Files[fileName] = StatusCreated
			modified = true
			continue
		}
		if dstFileMetadata.ContentHash != srcFileMetadata.ContentHash {
			dirSyncStruct.Files[fileName] = StatusModified
			modified = true
		}
		delete(dst.Files, fileName)
	}

	for fileName := range dst.Files {
		dirSyncStruct.Files[fileName] = StatusDeleted
		modified = true
	}

	for subdirName, srcSubdirStruct := range src.Subdirs {
		dstSubdirStruct, ok := dst.Subdirs[subdirName]
		if !ok {
			subdirSyncStruct := PreviewSync(srcSubdirStruct, dirmap.MakeEmptyDirStruct())
			subdirSyncStruct.Status = StatusCreated
			dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
			modified = true
			continue
		}
		subdirSyncStruct := PreviewSync(srcSubdirStruct, dstSubdirStruct)
		dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
		if !modified && subdirSyncStruct.Status != StatusNone {
			modified = true
		}
		delete(dst.Subdirs, subdirName)
	}

	for subdirName := range dst.Subdirs {
		subdirSyncStruct := PreviewSync(dirmap.MakeEmptyDirStruct(), dst.Subdirs[subdirName])
		subdirSyncStruct.Status = StatusDeleted
		dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
		modified = true
	}

	if modified {
		dirSyncStruct.Status = StatusModified
	}
	return dirSyncStruct
}

func SyncWithPreview(src, dst string, preview SyncPreview) error {
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
				err := os.Remove(dstPath)
				if err != nil {
					return err
				}
			}
		}
	case StatusDeleted:
		err := os.RemoveAll(dst)
		if err != nil {
			return err
		}
	}

	if preview.Status == StatusCreated || preview.Status == StatusModified {
		for subdir, subdirPreview := range preview.Subdirs {
			srcPath := filepath.Join(src, subdir)
			dstPath := filepath.Join(dst, subdir)
			err := SyncWithPreview(srcPath, dstPath, subdirPreview)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
