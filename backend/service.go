package backend

import (
	"log"
	"lsync/backend/internal/dirmap"
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
	srcDirStruct, err := dirmap.GetDirStruct(src)
	if err != nil {
		return lsync.SyncPreview{}, err
	}
	dstDirStruct, err := dirmap.GetDirStruct(dst)
	if err != nil {
		return lsync.SyncPreview{}, err
	}
	syncPreview := lsync.PreviewSync(srcDirStruct, dstDirStruct)
	log.Printf("Time taken (Seqeuntial): %v\n", time.Since(start))
	return syncPreview, nil
}

func (a *App) PreviewSyncAsync(src, dst string) (lsync.SyncPreview, error) {
	log.Println("Started Preview Sync Async")
	start := time.Now()
	srcCh, dstCh := make(chan dirmap.DirStructResult), make(chan dirmap.DirStructResult)
	go dirmap.GetDirStructAsync(src, src, srcCh, nil)
	go dirmap.GetDirStructAsync(dst, dst, dstCh, nil)
	srcRes, dstRes := <-srcCh, <-dstCh
	if srcRes.Err != nil {
		return lsync.SyncPreview{}, srcRes.Err
	}
	if dstRes.Err != nil {
		return lsync.SyncPreview{}, dstRes.Err
	}
	syncPreview := lsync.PreviewSync(srcRes.DirStruct, dstRes.DirStruct)
	log.Printf("Time taken (Async): %v\n", time.Since(start))
	return syncPreview, nil
}

func (a *App) SyncWithPreview(src, dst string, preview lsync.SyncPreview, ignoreDelete bool) error {
	return lsync.SyncWithPreview(src, dst, preview, ignoreDelete)
}
