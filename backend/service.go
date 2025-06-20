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
	log.Println("Started Preview Sync")
	srcRes := dirfetch.FetchDir(src)
	if srcRes.Err != nil {
		return lsync.SyncPreview{}, srcRes.Err
	}
	dstRes := dirfetch.FetchDir(dst)
	if dstRes.Err != nil {
		return lsync.SyncPreview{}, dstRes.Err
	}
	start := time.Now()
	preview := lsync.PreviewSync(srcRes.ItemMap, dstRes.ItemMap)
	log.Printf("Time taken for preview: %v\n", time.Since(start))
	return preview, nil
}

func (a *App) SyncWithPreview(src, dst string, preview lsync.SyncPreview, ignoreDelete bool) error {
	lsync.SyncWithPreview(src, dst, preview, ignoreDelete)
	return nil
}
