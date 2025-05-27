package backend

import (
	"log"
	"lsync/backend/internal/dirmap"
	"lsync/backend/pkg/sync"

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

func (a *App) PreviewSync(src, dst string) (sync.SyncPreview, error) {
	srcDirStruct, err := dirmap.GetDirStruct(src)
	if err != nil {
		return sync.SyncPreview{}, err
	}
	dstDirStruct, err := dirmap.GetDirStruct(dst)
	if err != nil {
		return sync.SyncPreview{}, err
	}
	return sync.PreviewSync(srcDirStruct, dstDirStruct), nil
}

func (a *App) SyncWithPreview(src, dst string, preview sync.SyncPreview, ignoreDelete bool) error {
	return sync.SyncWithPreview(src, dst, preview, ignoreDelete)
}
