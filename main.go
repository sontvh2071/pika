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

// Transparent space for panel shadows; config dimensions remain content size.
// Keep --shadow-margin in frontend/src/launcher.css in sync.
const shadowMargin = 48

func runWindow(app *App, cfg config.Config) error {
	// Install before Wails constructs/maps its GTK window, not in OnStartup
	// (which runs asynchronously and can miss the first native frame).
	installLauncherPresentation()
	return wails.Run(&options.App{
		Title: "Pika", Width: cfg.Window.Width + 2*shadowMargin, Height: cfg.Window.Height + 2*shadowMargin,
		MinWidth: 480 + 2*shadowMargin, MinHeight: 380 + 2*shadowMargin, Frameless: true, DisableResize: true,
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
