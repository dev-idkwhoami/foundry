package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"foundry/backend/features"
	"foundry/backend/transformer"
)

// ManualPatch holds info about a step the user needs to perform manually.
type ManualPatch struct {
	FeatureName string `json:"featureName"`
	FeatureID   string `json:"featureId"`
	File        string `json:"file"`
	Instruction string `json:"instruction"`
	Copy        string `json:"copy"`
}

// applyPatches applies auto patches in dependency order and returns manual patches.
func applyPatches(projectDir string, registry *features.Registry, selectedIDs []string, configValues map[string]string, emit func(string)) ([]ManualPatch, error) {
	// Get topological order (dependencies first).
	allSorted, err := registry.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("topological sort: %w", err)
	}

	// Filter to only selected features, preserving topo order.
	selected := make(map[string]bool)
	for _, id := range selectedIDs {
		selected[id] = true
	}

	var ordered []string
	for _, id := range allSorted {
		if selected[id] {
			ordered = append(ordered, id)
		}
	}

	var manualPatches []ManualPatch

	for _, featureID := range ordered {
		feature := registry.GetFeature(featureID)
		if feature == nil {
			continue
		}

		featureDir := filepath.Join(projectDir, "features", featureID)

		// Load mappings if they exist.
		var mappings []features.Mapping
		mappingsPath := filepath.Join(featureDir, "mappings.yaml")
		if _, err := os.Stat(mappingsPath); err == nil {
			mappings, err = features.ParseMappings(mappingsPath)
			if err != nil {
				return nil, fmt.Errorf("parsing mappings for %s: %w", featureID, err)
			}
		}

		// Extract config values scoped to this feature (featureId.key → key).
		featureConfig := make(map[string]string)
		prefix := featureID + "."
		for k, v := range configValues {
			if len(k) > len(prefix) && k[:len(prefix)] == prefix {
				featureConfig[k[len(prefix):]] = v
			}
		}

		// Pre-patch hooks for this feature.
		if len(feature.Hooks.PrePatch) > 0 {
			resolved := resolveCommands(featureID, feature.Hooks.PrePatch, configValues)
			if err := runCommands(projectDir, resolved, emit); err != nil {
				return nil, fmt.Errorf("pre-patch hook for %s: %w", featureID, err)
			}
		}

		for _, patch := range feature.Patches {
			mode := patch.Mode
			if mode == "" {
				mode = "auto"
			}

			if mode == "manual" {
				manualPatches = append(manualPatches, ManualPatch{
					FeatureName: feature.Name,
					FeatureID:   featureID,
					File:        patch.File,
					Instruction: patch.Instruction,
				})
				continue
			}

			// Auto patch: read diff, apply mappings, git apply.
			diffPath := filepath.Join(featureDir, patch.File)
			diffContent, err := os.ReadFile(diffPath)
			if err != nil {
				return nil, fmt.Errorf("reading patch %s/%s: %w", featureID, patch.File, err)
			}

			resolved := string(diffContent)
			if len(mappings) > 0 && len(featureConfig) > 0 {
				resolved, err = features.ResolveMappings(resolved, mappings, featureConfig)
				if err != nil {
					return nil, fmt.Errorf("resolving mappings for %s/%s: %w", featureID, patch.File, err)
				}
			}

			emit(fmt.Sprintf("Applying patch: %s/%s", featureID, patch.File))

			if err := gitApply(projectDir, resolved); err != nil {
				return nil, fmt.Errorf("git apply %s/%s: %w", featureID, patch.File, err)
			}
		}

		// Collect instructions for this feature, resolving tokens in both fields.
		for _, inst := range feature.Instructions {
			text := inst.Text
			copyText := inst.Copy
			if len(featureConfig) > 0 {
				if resolved, err := transformer.ResolveAll(text, featureConfig); err == nil {
					text = resolved
				}
				if copyText != "" {
					if resolved, err := transformer.ResolveAll(copyText, featureConfig); err == nil {
						copyText = resolved
					}
				}
			}
			manualPatches = append(manualPatches, ManualPatch{
				FeatureName: feature.Name,
				FeatureID:   featureID,
				Instruction: text,
				Copy:        copyText,
			})
		}

		// Post-patch hooks for this feature.
		if len(feature.Hooks.PostPatch) > 0 {
			resolved := resolveCommands(featureID, feature.Hooks.PostPatch, configValues)
			if err := runCommands(projectDir, resolved, emit); err != nil {
				return nil, fmt.Errorf("post-patch hook for %s: %w", featureID, err)
			}
		}
	}

	return manualPatches, nil
}

// gitApply applies a diff string to the project using git apply via stdin.
func gitApply(projectDir, diff string) error {
	cmd := exec.Command("git", "apply", "--whitespace=nowarn", "-")
	cmd.Dir = projectDir
	cmd.Stdin = strings.NewReader(diff)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return nil
}
