package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"pika/internal/config"
)

//go:embed all:frontend/dist
var assets embed.FS

func runWindow(app *App, cfg config.Config) error {
	return wails.Run(&options.App{
		Title: "Pika", Width: cfg.Window.Width, Height: cfg.Window.Height,
		MinWidth: 480, MinHeight: 380, Frameless: true, DisableResize: true,
		StartHidden: true, AlwaysOnTop: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: options.NewRGBA(0, 0, 0, 0),
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Linux:            &linux.Options{ProgramName: "pika", WindowIsTranslucent: true, WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand},
		Bind: []interface{}{
			app,
		},
	})
}
