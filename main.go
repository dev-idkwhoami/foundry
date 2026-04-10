package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"foundry/backend/appdata"
	"foundry/backend/config"
	"foundry/backend/db"
	foundrylog "foundry/backend/logger"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/build
var assets embed.FS

func main() {
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	debug := flag.Bool("debug", false, "Enable debug mode (shows developer tools in UI)")
	flag.Parse()

	if os.Getenv("FOUNDRY_VERBOSE") == "1" {
		*verbose = true
	}

	// Positional arg: project name
	projectName := ""
	if args := flag.Args(); len(args) > 0 {
		projectName = args[0]
	}

	if err := appdata.Init(); err != nil {
		log.Fatalf("Failed to initialize app data: %v", err)
	}

	if err := db.Open(); err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger, err := foundrylog.New(*verbose)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		log.Fatalf("Failed to load config: %v", err)
	}

	startupCtx := buildStartupContext(projectName, logger)

	app := NewApp(cfg, logger, startupCtx, *debug)

	err = wails.Run(&options.App{
		Title:     "Foundry",
		Width:     1024,
		Height:    768,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 27, B: 27, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatalf("Error running app: %v", err)
	}
}

// buildStartupContext determines if the app was started from a terminal with
// a valid working directory, or from a context-less launcher (Start Menu, etc.).
func buildStartupContext(projectName string, logger *foundrylog.Logger) *StartupContext {
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get working directory: %v", err)
		return &StartupContext{}
	}

	hasContext := isValidWorkingDir(cwd)

	logger.Info("CWD: %s, hasContext: %v, projectName: %s", cwd, hasContext, projectName)

	ctx := &StartupContext{
		ProjectName: projectName,
		HasContext:   hasContext,
	}

	if hasContext {
		ctx.WorkingDir = cwd
	}

	return ctx
}

// isValidWorkingDir returns false if CWD looks like a system directory
// or the app's own install location (indicating a Start Menu / double-click launch).
func isValidWorkingDir(cwd string) bool {
	normalized := strings.ToLower(filepath.Clean(cwd))

	systemPaths := []string{
		strings.ToLower(filepath.Clean(os.Getenv("WINDIR"))),
		strings.ToLower(filepath.Clean(os.Getenv("SYSTEMROOT"))),
	}

	for _, sys := range systemPaths {
		if sys != "" && strings.HasPrefix(normalized, sys) {
			return false
		}
	}

	// Check if CWD is the app's own directory
	exe, err := os.Executable()
	if err == nil {
		exeDir := strings.ToLower(filepath.Dir(filepath.Clean(exe)))
		if normalized == exeDir {
			return false
		}
	}

	return true
}
