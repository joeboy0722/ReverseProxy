package main

import (
	"embed"
	"flag"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 解析命令列旗標
	cliMode := flag.Bool("cli", false, "Start in interactive CLI mode")
	cliShort := flag.Bool("c", false, "Start in interactive CLI mode (short)")
	flag.Parse()

	// Create an instance of the app structure
	app := NewApp()

	if *cliMode || *cliShort {
		showConsole()
		runCLI(app, flag.Args())
		return
	}

	hideConsole()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "reverse-proxy",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
