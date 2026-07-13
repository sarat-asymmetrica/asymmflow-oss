package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// brandWindowTitle is the desktop window title shown in the OS title bar and
// alt-tab. Branding is configuration, not code (repo law): the committed value
// is the SYNTHETIC product identity, and a deployment overrides it at BUILD
// time with no source edit —
//
//	wails build -ldflags "-X 'main.brandWindowTitle=Your Product'"
//
// The `wails.json` name field controls the binary/installer name; this controls
// the runtime window title. See docs/DEPLOYMENT_BRANDING.md slot 4.
var brandWindowTitle = "AsymmFlow"

func main() {
	resetStartupDiagnostics()
	appendStartupDiagnostic("MAIN: starting Wails runtime")

	// Create an instance of the app structure
	app := NewApp()

	// Domain service bindings delegate to App while giving the frontend a v3-ready API surface.
	financeService := NewFinanceService(app)
	crmService := NewCRMService(app)
	butlerService := NewButlerService(app)
	documentsService := NewDocumentsService(app)
	syncService := NewSyncServiceBinding(app)
	infraService := NewInfraService(app)

	// Create application with options
	err := wails.Run(&options.App{
		Title:            brandWindowTitle,
		Width:            1400,
		Height:           900,
		WindowStartState: options.Maximised,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 253, G: 251, B: 247, A: 1}, // Wabi-Sabi paper cream
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown, // Graceful cleanup
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
		Bind: []any{
			app,
			financeService,
			crmService,
			butlerService,
			documentsService,
			syncService,
			infraService,
		},
	})

	if err != nil {
		appendStartupDiagnostic("MAIN: Wails runtime returned error: %v", err)
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}
}
