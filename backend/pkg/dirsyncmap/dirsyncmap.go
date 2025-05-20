package dirsyncmap

import (
	"lsync/backend/internal/dirmap"
)

type SyncStatus string

const (
	StatusNone     SyncStatus = "None"
	StatusCreated  SyncStatus = "Created"
	StatusModified SyncStatus = "Modified"
	StatusDeleted  SyncStatus = "Deleted"
)

type DirSyncStruct struct {
	Status  SyncStatus
	Subdirs map[string]DirSyncStruct
	Files   map[string]SyncStatus
}

func GetDirSyncStruct(src, dst dirmap.DirStruct) DirSyncStruct {
	dirSyncStruct := DirSyncStruct{
		Status:  StatusNone,
		Subdirs: make(map[string]DirSyncStruct),
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
			subdirSyncStruct := GetDirSyncStruct(srcSubdirStruct, dirmap.MakeEmptyDirStruct())
			subdirSyncStruct.Status = StatusCreated
			dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
			modified = true
			continue
		}
		subdirSyncStruct := GetDirSyncStruct(srcSubdirStruct, dstSubdirStruct)
		dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
		if !modified && subdirSyncStruct.Status != StatusNone {
			modified = true
		}
		delete(dst.Subdirs, subdirName)
	}

	for subdirName := range dst.Subdirs {
		subdirSyncStruct := GetDirSyncStruct(dirmap.MakeEmptyDirStruct(), dst.Subdirs[subdirName])
		subdirSyncStruct.Status = StatusDeleted
		dirSyncStruct.Subdirs[subdirName] = subdirSyncStruct
		modified = true
	}

	if modified {
		dirSyncStruct.Status = StatusModified
	}
	return dirSyncStruct
}
