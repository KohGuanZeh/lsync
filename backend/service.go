package backend

import (
	"fmt"
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
	srcCh := dirfetch.FetchDirAsync(src)
	dstCh := dirfetch.FetchDirAsync(dst)
	srcDirTree, dstDirTree := <-srcCh, <-dstCh
	if srcDirTree == nil {
		return lsync.SyncPreview{}, fmt.Errorf("failed to get directory structure for src: %s", src)
	}
	if dstDirTree == nil {
		return lsync.SyncPreview{}, fmt.Errorf("failed to get directory structure for dst: %s", dst)
	}
	log.Printf("Time taken for dirfetch: %v\n", time.Since(start))
	return lsync.SyncPreview{}, nil
}

func (a *App) SyncWithPreview(src, dst string, preview lsync.SyncPreview, ignoreDelete bool) error {
	return lsync.SyncWithPreview(src, dst, preview, ignoreDelete)
}
