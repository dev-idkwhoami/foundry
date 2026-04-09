package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"foundry/backend/config"
	"foundry/backend/db"
	"foundry/backend/features"
	"foundry/backend/git"
	"foundry/backend/herd"
	foundrylog "foundry/backend/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// stageLabels maps stage IDs to human-readable names for log banners.
var stageLabels = map[string]string{
	StagePreClone:    "Pre-Clone Hooks",
	StageClone:       "Clone",
	StagePostClone:   "Post-Clone Hooks",
	StagePreHerd:     "Pre-Herd Hooks",
	StageHerd:        "Herd Setup",
	StagePostHerd:    "Post-Herd Hooks",
	StagePatching:    "Patching",
	StagePreInstall:  "Pre-Install Hooks",
	StagePostInstall: "Setup Commands",
	StagePreCleanup:  "Pre-Cleanup Hooks",
	StageCleanup:     "Cleanup",
	StagePostCleanup: "Post-Cleanup Hooks",
}

// InstallRequest contains all parameters needed to run the installation.
type InstallRequest struct {
	ProjectName   string            `json:"projectName"`
	WorkingDir    string            `json:"workingDir"`
	SelectedIDs   []string          `json:"selectedIds"`
	ConfigValues  map[string]string `json:"configValues"`
	TempClonePath string            `json:"tempClonePath"`
}

// InstallResult is returned when the installation completes.
type InstallResult struct {
	Success      bool          `json:"success"`
	ManualSteps  []ManualPatch `json:"manualSteps"`
	ErrorMessage string        `json:"errorMessage"`
	ErrorStage   string        `json:"errorStage"`
}

// stages for progress tracking.
const (
	StagePreClone    = "pre-clone"
	StageClone       = "clone"
	StagePostClone   = "post-clone"
	StagePreHerd     = "pre-herd"
	StageHerd        = "herd"
	StagePostHerd    = "post-herd"
	StagePatching    = "patching"
	StagePreInstall  = "pre-install"
	StagePostInstall = "post-install"
	StagePreCleanup  = "pre-cleanup"
	StageCleanup     = "cleanup"
	StagePostCleanup = "post-cleanup"
)

// Run executes the full installation pipeline.
func Run(ctx context.Context, req InstallRequest, cfg *config.AppConfig, registry *features.Registry, logger *foundrylog.Logger) InstallResult {
	projectDir := fmt.Sprintf("%s\\%s", req.WorkingDir, req.ProjectName)

	emitLog := func(msg string) {
		logger.Info(msg)
		runtime.EventsEmit(ctx, "install:log", map[string]string{
			"message": msg,
			"level":   "info",
		})
	}

	emitProgress := func(stage, status string) {
		if status == "running" {
			label := stageLabels[stage]
			if label == "" {
				label = stage
			}
			banner := fmt.Sprintf("── %s %s", label, strings.Repeat("─", max(0, 40-len(label))))
			emitLog(banner)
		}
		runtime.EventsEmit(ctx, "install:progress", map[string]string{
			"stage":  stage,
			"status": status,
		})
	}

	fail := func(stage, msg string) InstallResult {
		logger.Error("Install failed at %s: %s", stage, msg)
		runtime.EventsEmit(ctx, "install:error", map[string]string{
			"stage":   stage,
			"message": msg,
		})
		return InstallResult{
			Success:      false,
			ErrorMessage: msg,
			ErrorStage:   stage,
		}
	}

	// Helper to run a hook stage.
	hook := func(stage string, hookFn func(*features.Feature) []string, dir string) error {
		emitProgress(stage, "running")
		if err := runHook(dir, registry, req.SelectedIDs, req.ConfigValues, hookFn, emitLog); err != nil {
			return fmt.Errorf("%s", err.Error())
		}
		emitProgress(stage, "done")
		return nil
	}

	// 1. pre-clone hooks (run from working dir — project doesn't exist yet).
	if err := hook(StagePreClone, func(f *features.Feature) []string { return f.Hooks.PreClone }, req.WorkingDir); err != nil {
		return fail(StagePreClone, err.Error())
	}

	// 2. Clone repository to target directory.
	emitProgress(StageClone, "running")
	emitLog("Cloning repository...")
	if err := git.CloneToTarget(cfg.Repository, req.WorkingDir, req.ProjectName); err != nil {
		return fail(StageClone, err.Error())
	}
	emitProgress(StageClone, "done")

	// 3. post-clone hooks.
	if err := hook(StagePostClone, func(f *features.Feature) []string { return f.Hooks.PostClone }, projectDir); err != nil {
		return fail(StagePostClone, err.Error())
	}

	// 4. pre-herd hooks.
	if err := hook(StagePreHerd, func(f *features.Feature) []string { return f.Hooks.PreHerd }, projectDir); err != nil {
		return fail(StagePreHerd, err.Error())
	}

	// 5. Herd site setup (link, database, .env) + auth.json.
	emitProgress(StageHerd, "running")
	emitLog("Setting up Herd site...")
	if err := herd.Setup(projectDir, req.ProjectName); err != nil {
		return fail(StageHerd, err.Error())
	}
	emitLog("Configuring Composer authentication...")
	if err := writeAuthJSON(projectDir, cfg); err != nil {
		return fail(StageHerd, err.Error())
	}
	emitProgress(StageHerd, "done")

	// 6. post-herd hooks.
	if err := hook(StagePostHerd, func(f *features.Feature) []string { return f.Hooks.PostHerd }, projectDir); err != nil {
		return fail(StagePostHerd, err.Error())
	}

	// 7. Patching (with per-feature pre/post-patch hooks).
	emitProgress(StagePatching, "running")
	emitLog("Applying feature patches...")
	manualPatches, err := applyPatches(projectDir, registry, req.SelectedIDs, req.ConfigValues, emitLog)
	if err != nil {
		return fail(StagePatching, err.Error())
	}
	emitProgress(StagePatching, "done")

	// 8. pre-install hooks.
	if err := hook(StagePreInstall, func(f *features.Feature) []string { return f.Hooks.PreInstall }, projectDir); err != nil {
		return fail(StagePreInstall, err.Error())
	}

	// 9. Global post-install commands.
	emitProgress(StagePostInstall, "running")
	emitLog("Running post-install commands...")
	if err := runCommands(projectDir, cfg.Setup, emitLog); err != nil {
		return fail(StagePostInstall, err.Error())
	}
	emitProgress(StagePostInstall, "done")

	// 10. post-install hooks.
	if err := hook("hooks:post-install", func(f *features.Feature) []string { return f.Hooks.PostInstall }, projectDir); err != nil {
		return fail("hooks:post-install", err.Error())
	}

	// 11. pre-cleanup hooks.
	if err := hook(StagePreCleanup, func(f *features.Feature) []string { return f.Hooks.PreCleanup }, projectDir); err != nil {
		return fail(StagePreCleanup, err.Error())
	}

	// 12. Cleanup.
	emitProgress(StageCleanup, "running")
	emitLog("Cleaning up...")
	if err := cleanup(projectDir, req.TempClonePath, cfg.Cleanup, emitLog); err != nil {
		return fail(StageCleanup, err.Error())
	}
	emitProgress(StageCleanup, "done")

	// 13. post-cleanup hooks.
	if err := hook(StagePostCleanup, func(f *features.Feature) []string { return f.Hooks.PostCleanup }, projectDir); err != nil {
		return fail(StagePostCleanup, err.Error())
	}

	// Save working dir as recent.
	cfg.AddRecentDirectory(req.WorkingDir)
	_ = cfg.Save()

	// Record installation in tracking database.
	recordInstallation(projectDir, req, cfg, registry, emitLog)

	runtime.EventsEmit(ctx, "install:complete", nil)

	return InstallResult{
		Success:     true,
		ManualSteps: manualPatches,
	}
}

// recordInstallation saves the installation and its features to the tracking database.
func recordInstallation(projectDir string, req InstallRequest, cfg *config.AppConfig, registry *features.Registry, emitLog func(string)) {
	inst := db.Installation{
		ProjectPath: projectDir,
		ProjectName: req.ProjectName,
		Repository:  cfg.Repository,
		SiteName:    req.ProjectName,
		DbName:      req.ProjectName,
	}

	installID, err := db.RecordInstallation(inst)
	if err != nil {
		emitLog(fmt.Sprintf("Warning: failed to record installation: %v", err))
		return
	}

	var dbFeatures []db.InstalledFeature
	for _, featureID := range req.SelectedIDs {
		f := registry.GetFeature(featureID)
		name := featureID
		if f != nil {
			name = f.Name
		}

		// Collect config values scoped to this feature as JSON.
		featureConfig := make(map[string]string)
		prefix := featureID + "."
		for k, v := range req.ConfigValues {
			if strings.HasPrefix(k, prefix) {
				featureConfig[k[len(prefix):]] = v
			}
		}
		configJSON, _ := json.Marshal(featureConfig)

		dbFeatures = append(dbFeatures, db.InstalledFeature{
			InstallationID: installID,
			FeatureID:      featureID,
			FeatureName:    name,
			ConfigValues:   string(configJSON),
		})
	}

	if err := db.RecordFeatures(installID, dbFeatures); err != nil {
		emitLog(fmt.Sprintf("Warning: failed to record features: %v", err))
	}
}

// writeAuthJSON creates auth.json with Flux credentials if available.
func writeAuthJSON(projectDir string, cfg *config.AppConfig) error {
	username := cfg.FluxUsername
	password := cfg.FluxLicenseKey
	if username == "" || password == "" {
		return nil
	}

	composerURL := cfg.FluxComposerURL
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
