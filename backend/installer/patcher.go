package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"foundry/backend/executil"
	"foundry/backend/features"
	"foundry/backend/patcher"
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
	var cdiffDiffs []patcher.Diff

	// First pass: collect all patches, run pre-patch hooks, gather cdiffs.
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

			// Auto patch: read diff, apply mappings.
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

			if patch.Format == "cdiff" {
				// Parse and collect for batch apply.
				diff, err := patcher.Parse(resolved)
				if err != nil {
					return nil, fmt.Errorf("parsing cdiff %s/%s: %w", featureID, patch.File, err)
				}
				// Tag all hunks with the feature ID.
				for i := range diff.Files {
					for j := range diff.Files[i].Hunks {
						diff.Files[i].Hunks[j].FeatureID = featureID
					}
				}
				cdiffDiffs = append(cdiffDiffs, *diff)
			} else {
				// Legacy git apply.
				emit(fmt.Sprintf("Applying patch: %s/%s", featureID, patch.File))
				if err := gitApply(projectDir, resolved); err != nil {
					return nil, fmt.Errorf("git apply %s/%s: %w", featureID, patch.File, err)
				}
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
	}

	// Apply all cdiff patches in one merged pass.
	if len(cdiffDiffs) > 0 {
		emit("Applying contextual diffs...")
		result, err := patcher.Apply(patcher.ApplyRequest{
			ProjectDir: projectDir,
			Diffs:      cdiffDiffs,
		})
		if err != nil {
			return nil, fmt.Errorf("applying cdiffs: %w", err)
		}
		if len(result.Conflicts) > 0 {
			var msgs []string
			for _, c := range result.Conflicts {
				msgs = append(msgs, fmt.Sprintf("%s: %s vs %s — %s", c.File, c.FeatureA, c.FeatureB, c.Reason))
			}
			return nil, fmt.Errorf("patch conflicts:\n%s", strings.Join(msgs, "\n"))
		}
		for _, f := range result.Modified {
			emit(fmt.Sprintf("Modified: %s", f))
		}
	}

	// Run post-patch hooks after all patches are applied.
	for _, featureID := range ordered {
		feature := registry.GetFeature(featureID)
		if feature == nil {
			continue
		}
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
	cmd := executil.Command("git", "apply", "--whitespace=nowarn", "-")
	cmd.Dir = projectDir
	cmd.Stdin = strings.NewReader(diff)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return nil
}
