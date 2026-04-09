package installer

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"foundry/backend/features"
	"foundry/backend/transformer"
)

// runHook runs a specific hook phase for all selected features in topo order.
// hookFn extracts the relevant command list from the feature's Hooks struct.
func runHook(projectDir string, registry *features.Registry, selectedIDs []string, configValues map[string]string, hookFn func(*features.Feature) []string, emit func(string)) error {
	ordered := topoFilterSelected(registry, selectedIDs)

	for _, id := range ordered {
		f := registry.GetFeature(id)
		if f == nil {
			continue
		}
		commands := hookFn(f)
		if len(commands) == 0 {
			continue
		}
		resolved := resolveCommands(id, commands, configValues)
		if err := runCommands(projectDir, resolved, emit); err != nil {
			return fmt.Errorf("hook for %s: %w", id, err)
		}
	}
	return nil
}

// topoFilterSelected returns selected feature IDs in topological order.
func topoFilterSelected(registry *features.Registry, selectedIDs []string) []string {
	allSorted, _ := registry.TopologicalSort()
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
	return ordered
}

// resolveCommands resolves {{key:transformer}} tokens in command strings
// using the config values scoped to the given feature ID.
func resolveCommands(featureID string, commands []string, configValues map[string]string) []string {
	featureConfig := make(map[string]string)
	prefix := featureID + "."
	for k, v := range configValues {
		if strings.HasPrefix(k, prefix) {
			featureConfig[k[len(prefix):]] = v
		}
	}

	if len(featureConfig) == 0 {
		return commands
	}

	resolved := make([]string, len(commands))
	for i, cmd := range commands {
		r, err := transformer.ResolveAll(cmd, featureConfig)
		if err == nil {
			resolved[i] = r
		} else {
			resolved[i] = cmd
		}
	}
	return resolved
}

// runCommands executes each command sequentially in the project directory,
// streaming stdout/stderr line by line via the emit callback.
func runCommands(projectDir string, commands []string, emit func(string)) error {
	for _, raw := range commands {
		emit(fmt.Sprintf("$ %s", raw))

		parts := strings.Fields(raw)
		if len(parts) == 0 {
			continue
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = projectDir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("stdout pipe for %q: %w", raw, err)
		}

		cmd.Stderr = cmd.Stdout // merge stderr into stdout

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("starting %q: %w", raw, err)
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			emit(scanner.Text())
		}

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("command %q failed: %w", raw, err)
		}
	}

	return nil
}
