package backend

import (
	"log"
	"lsync/backend/internal/dirfetch"
	"lsync/backend/pkg/lsync"
	"time"

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

func (a *App) PreviewSync(src, dst string) (lsync.SyncPreview, error) {
	log.Println("Started Preview Sync Async")
	start := time.Now()
	srcDirStruct := dirfetch.FetchDir(src)
	dstDirStruct := dirfetch.FetchDir(dst)
	syncPreview := lsync.PreviewSync(&srcDirStruct, &dstDirStruct)
	log.Printf("Time taken (Async): %v\n", time.Since(start))
	return syncPreview, nil
}

func (a *App) SyncWithPreview(src, dst string, preview lsync.SyncPreview, ignoreDelete bool) error {
	return lsync.SyncWithPreview(src, dst, preview, ignoreDelete)
}
