package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"foundry/backend/config"
	"foundry/backend/features"
	"foundry/backend/git"
	"foundry/backend/installer"
	foundrylog "foundry/backend/logger"
	"foundry/backend/transformer"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type StartupContext struct {
	ProjectName string `json:"projectName"`
	WorkingDir  string `json:"workingDir"`
	HasContext  bool   `json:"hasContext"`
}

type CompatResult struct {
	Compatible bool   `json:"compatible"`
	Reason     string `json:"reason"`
}

type StartupResult struct {
	Done    bool   `json:"done"`
	Error   string `json:"error"`
}

type App struct {
	ctx           context.Context
	config        *config.AppConfig
	logger        *foundrylog.Logger
	startupCtx    *StartupContext
	registry      *features.Registry
	tempClonePath string
	patchMu       sync.Mutex
	startupResult StartupResult
	debug         bool
}

func NewApp(cfg *config.AppConfig, logger *foundrylog.Logger, startupCtx *StartupContext, debug bool) *App {
	return &App{
		config:     cfg,
		logger:     logger,
		startupCtx: startupCtx,
		debug:      debug,
	}
}

func (a *App) IsDebug() bool {
	return a.debug
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.logger.SetContext(ctx)
	a.logger.Info("Foundry started")

	go func() {
		if a.config.Repository == "" {
			a.startupResult = StartupResult{Done: true, Error: "repository is not configured"}
			runtime.EventsEmit(a.ctx, "ready", "repository is not configured")
			return
		}

		clonedPath, err := git.CloneOrPullTemp(a.config.Repository)
		if err != nil {
			a.startupResult = StartupResult{Done: true, Error: err.Error()}
			runtime.EventsEmit(a.ctx, "clone:error", err.Error())
			return
		}

		registry, err := features.BuildRegistry(clonedPath)
		if err != nil {
			a.startupResult = StartupResult{Done: true, Error: err.Error()}
			runtime.EventsEmit(a.ctx, "registry:error", err.Error())
			return
		}

		a.registry = registry
		a.tempClonePath = clonedPath
		a.startupResult = StartupResult{Done: true}
		runtime.EventsEmit(a.ctx, "ready", "")
	}()
}

func (a *App) GetStartupResult() StartupResult {
	return a.startupResult
}

func (a *App) GetConfig() *config.AppConfig {
	return a.config
}

func (a *App) GetStartupContext() *StartupContext {
	return a.startupCtx
}

func (a *App) GetRecentDirectories() []string {
	return a.config.RecentDirectories
}

func (a *App) AddRecentDirectory(dir string) error {
	a.config.AddRecentDirectory(dir)
	return a.config.Save()
}

// GetComposerVersion returns the installed Composer version string, or an
// error if Composer is not found.
func (a *App) GetComposerVersion() (string, error) {
	out, err := exec.Command("composer", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("composer not found: %w", err)
	}
	// Output is like "Composer version 2.7.1 2024-02-09 ..."
	parts := strings.Fields(string(out))
	for i, p := range parts {
		if p == "version" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}

// GetFluxLicenseKey returns the stored Flux UI Pro license key.
func (a *App) GetFluxLicenseKey() string {
	return a.config.FluxLicenseKey
}

// SetFluxLicenseKey saves a Flux UI Pro license key to config.
func (a *App) SetFluxLicenseKey(key string) error {
	a.config.FluxLicenseKey = strings.TrimSpace(key)
	return a.config.Save()
}

// GetFluxUsername returns the stored Flux UI Pro username.
func (a *App) GetFluxUsername() string {
	return a.config.FluxUsername
}

// SetFluxUsername saves a Flux UI Pro username to config.
func (a *App) SetFluxUsername(username string) error {
	a.config.FluxUsername = strings.TrimSpace(username)
	return a.config.Save()
}

// WriteAuthJSON creates a Composer auth.json in projectDir with Flux
// credentials. Returns nil without writing if credentials are incomplete.
func (a *App) WriteAuthJSON(projectDir string) error {
	username := a.config.FluxUsername
	password := a.config.FluxLicenseKey
	if username == "" || password == "" {
		return nil
	}

	composerURL := a.config.FluxComposerURL
	if composerURL == "" {
		composerURL = "composer.fluxui.dev"
	}

	authData := map[string]any{
		"http-basic": map[string]any{
			composerURL: map[string]string{
				"username": username,
				"password": password,
			},
		},
	}

	data, err := json.MarshalIndent(authData, "", "    ")
	if err != nil {
		return fmt.Errorf("marshalling auth.json: %w", err)
	}

	authPath := filepath.Join(projectDir, "auth.json")
	if err := os.WriteFile(authPath, data, 0644); err != nil {
		return fmt.Errorf("writing auth.json: %w", err)
	}

	return nil
}

// SetRepository updates the repository URL in config.
func (a *App) SetRepository(url string) error {
	a.config.Repository = strings.TrimSpace(url)
	return a.config.Save()
}

func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select project directory",
	})
}

func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

// OpenInExplorer opens the given directory in Windows Explorer.
func (a *App) OpenInExplorer(path string) error {
	cmd := exec.Command("explorer", path)
	return cmd.Start()
}

// OpenFileInEditor opens the given file in the system default editor.
func (a *App) OpenFileInEditor(path string) error {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	return cmd.Start()
}

// CheckTargetDirectory checks if the target directory exists and is non-empty.
// Returns "empty" if it doesn't exist or is empty, "not-empty" if it has files.
func (a *App) CheckTargetDirectory(path string) string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "empty" // doesn't exist or can't read = fine to use
	}
	if len(entries) == 0 {
		return "empty"
	}
	return "not-empty"
}

// GetFeatures returns all features from the registry, or nil if the
// registry has not been built yet.
func (a *App) GetFeatures() []*features.Feature {
	if a.registry == nil {
		return nil
	}
	return a.registry.Features
}

// GetFeatureRegistry returns the full feature registry, or nil if the
// registry has not been built yet.
func (a *App) GetFeatureRegistry() *features.Registry {
	if a.registry == nil {
		return nil
	}
	return a.registry
}

// GetGitVersion returns the installed Git version string, or an error if
// Git is not found.
func (a *App) GetGitVersion() (string, error) {
	out, err := exec.Command("git", "version").Output()
	if err != nil {
		return "", fmt.Errorf("git not found: %w", err)
	}
	// Output is like "git version 2.45.1.windows.1"
	parts := strings.Fields(string(out))
	if len(parts) >= 3 {
		return parts[2], nil
	}
	return strings.TrimSpace(string(out)), nil
}

// GetHerdVersion returns the installed Herd version string, or an error if
// Herd is not found.
func (a *App) GetHerdVersion() (string, error) {
	out, err := exec.Command("herd", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("herd not found: %w", err)
	}
	// Output is like "Herd 1.28.0"
	parts := strings.Fields(string(out))
	if len(parts) >= 2 {
		return parts[1], nil
	}
	return strings.TrimSpace(string(out)), nil
}

// GetPHPVersion returns the installed PHP version string, or an error if
// PHP is not found.
func (a *App) GetPHPVersion() (string, error) {
	out, err := exec.Command("php", "-v").Output()
	if err != nil {
		return "", fmt.Errorf("php not found: %w", err)
	}
	// First line is like "PHP 8.3.4 (cli) ..."
	firstLine := strings.SplitN(string(out), "\n", 2)[0]
	parts := strings.Fields(firstLine)
	if len(parts) >= 2 {
		return parts[1], nil
	}
	return strings.TrimSpace(firstLine), nil
}

// CheckPatchCompatibility checks whether a feature's auto patches can be
// applied on top of the currently selected features. It applies selected
// features' patches to the temp clone, runs git apply --check for the
// candidate, then resets the clone.
func (a *App) CheckPatchCompatibility(featureID string, selectedIDs []string) CompatResult {
	a.patchMu.Lock()
	defer a.patchMu.Unlock()

	if a.registry == nil || a.tempClonePath == "" {
		return CompatResult{false, "registry not ready"}
	}

	// Reset temp clone to clean state first.
	resetCmd := exec.Command("git", "checkout", ".")
	resetCmd.Dir = a.tempClonePath
	_ = resetCmd.Run()

	// Apply all currently selected features' auto patches.
	for _, sid := range selectedIDs {
		f := a.registry.GetFeature(sid)
		if f == nil {
			continue
		}
		for _, p := range f.Patches {
			patchPath := filepath.Join(a.tempClonePath, "features", sid, p.File)
			cmd := exec.Command("git", "apply", patchPath)
			cmd.Dir = a.tempClonePath
			_ = cmd.Run() // best-effort; these should succeed
		}
	}

	// Now check if the candidate feature's patches apply cleanly.
	candidate := a.registry.GetFeature(featureID)
	if candidate == nil {
		return CompatResult{false, fmt.Sprintf("feature %q not found", featureID)}
	}

	for _, p := range candidate.Patches {
		patchPath := filepath.Join(a.tempClonePath, "features", featureID, p.File)
		cmd := exec.Command("git", "apply", "--check", patchPath)
		cmd.Dir = a.tempClonePath
		if out, err := cmd.CombinedOutput(); err != nil {
			// Reset before returning.
			r := exec.Command("git", "checkout", ".")
			r.Dir = a.tempClonePath
			_ = r.Run()
			return CompatResult{false, strings.TrimSpace(string(out))}
		}
	}

	// Reset the temp clone back to clean state.
	resetCmd2 := exec.Command("git", "checkout", ".")
	resetCmd2.Dir = a.tempClonePath
	_ = resetCmd2.Run()

	return CompatResult{true, ""}
}

// ResolveToken applies a chain of transformers to a value and returns
// the result. Used by the frontend for live preview of config values.
func (a *App) ResolveToken(value string, transforms []string) string {
	for _, name := range transforms {
		fn, ok := transformer.Registry[name]
		if !ok {
			return value
		}
		value = fn(value)
	}
	return value
}

// Install runs the full installation pipeline asynchronously and streams
// progress events to the frontend.
func (a *App) Install(req installer.InstallRequest) {
	req.TempClonePath = a.tempClonePath

	go func() {
		result := installer.Run(a.ctx, req, a.config, a.registry, a.logger)
		runtime.EventsEmit(a.ctx, "install:result", result)
	}()
}
