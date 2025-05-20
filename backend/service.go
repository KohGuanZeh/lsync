package backend

import (
	"log"
	"lsync/backend/internal/dirmap"
	"lsync/backend/pkg/dirsyncmap"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SelectDirectory(title string) string {
	options := runtime.OpenDialogOptions{
		Title: title,
	}
	dir, err := runtime.OpenDirectoryDialog(a.Ctx, options)
	if err != nil {
		log.Println(err)
		return ""
	}
	return dir
}

func (a *App) PreviewSync(src, dst string) (dirsyncmap.DirSyncStruct, error) {
	srcDirStruct, err := dirmap.GetDirStruct(src)
	if err != nil {
		return dirsyncmap.DirSyncStruct{}, err
	}
	dstDirStruct, err := dirmap.GetDirStruct(dst)
	if err != nil {
		return dirsyncmap.DirSyncStruct{}, err
	}
	return dirsyncmap.GetDirSyncStruct(srcDirStruct, dstDirStruct), nil
}
