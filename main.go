package main

import (
	"embed"
	"log"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// assets holds the compiled React/TypeScript frontend.
// Wails embeds this into the binary so the app runs without external files.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Structured text logs to stderr — visible in the `wails dev` terminal.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "AresSim",
		Width:            1440,
		Height:           900,
		MinWidth:         1024,
		MinHeight:        768,
		DisableResize:    false,
		Fullscreen:       false,
		Frameless:        false,
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 21, A: 255},

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		// Wails lifecycle hooks wired to App methods.
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		OnDomReady: app.domReady,

		// All public methods on App are exposed to the frontend
		// as window.go.main.App.MethodName() via the generated wailsjs bindings.
		Bind: []interface{}{
			app,
		},

		// Platform-specific options.
		// Enable browser DevTools in dev builds (right-click → Inspect, or Cmd+Option+I).
		EnableDefaultContextMenu: true,

		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "AresSim",
				Message: "Operational & Strategic Wargame Engine",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Linux: &linux.Options{
			ProgramName: "AresSim",
		},
	})

	if err != nil {
		log.Fatalf("wails.Run: %v", err)
	}
}
