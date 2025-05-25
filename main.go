package main

import (
	"embed"
	"lsync/backend"
	"lsync/backend/pkg/sync"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := backend.NewApp()
	dirSyncStruct := sync.SyncPreview{}
	var dirSyncStatus = []struct {
		Value  sync.SyncStatus
		TSName string
	}{
		{sync.StatusNone, "None"},
		{sync.StatusCreated, "Created"},
		{sync.StatusModified, "Modified"},
		{sync.StatusDeleted, "Deleted"},
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Lsync",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
			&dirSyncStruct,
		},
		EnumBind: []interface{}{
			dirSyncStatus,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
