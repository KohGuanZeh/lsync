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
	start := time.Now()
	srcCh := dirfetch.FetchDirAsync(src)
	dstCh := dirfetch.FetchDirAsync(dst)
	srcRes, dstRes := <-srcCh, <-dstCh
	if srcRes.Err != nil {
		return lsync.SyncPreview{}, srcRes.Err
	}
	if dstRes.Err != nil {
		return lsync.SyncPreview{}, dstRes.Err
	}
	log.Printf("Time taken for dirfetch: %v\n", time.Since(start))

	start = time.Now()
	preview := lsync.PreviewSync(srcRes.ItemMap, dstRes.ItemMap)
	log.Printf("Time taken for preview: %v\n", time.Since(start))
	return preview, nil
}

func (a *App) SyncWithPreview(src, dst string, preview lsync.SyncPreview, ignoreDelete bool) error {
	log.Println("Synced...")
	return lsync.SyncWithPreview(src, dst, preview, ignoreDelete)
}
