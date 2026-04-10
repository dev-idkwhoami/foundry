package cli

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"foundry/backend/executil"
	"foundry/backend/transformer"
)

// runCreate scaffolds a new feature: creates a git branch, the feature
// directory, and minimal manifest.yaml + mappings.yaml files.
//
// Usage: foundry-cli create [<name>]
//
// If <name> is omitted, the user is prompted. The feature ID is derived
// from the name by applying snake_case + lowercase (e.g. "Themes" → "themes",
// "Magic Link" → "magic_link").
func runCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	name := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if name == "" {
		fmt.Print("Feature name: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			name = strings.TrimSpace(scanner.Text())
		}
		if name == "" {
			return fmt.Errorf("feature name is required")
		}
	}

	// Derive ID: snake_case + lowercase.
	toSnake := transformer.Registry["snake"]
	toLower := transformer.Registry["lower"]
	id := toLower(toSnake(name))

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	featDir := filepath.Join(cwd, "features", id)

	// Guard: don't overwrite an existing feature.
	if _, err := os.Stat(featDir); err == nil {
		return fmt.Errorf("feature directory already exists: %s", featDir)
	}

	// Create git branch.
	branch := "feature/" + id
	fmt.Printf("Creating branch %s... ", branch)
	cmd := executil.Command("git", "checkout", "-b", branch)
	cmd.Dir = cwd
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("creating branch: %s", strings.TrimSpace(string(out)))
	}
	fmt.Println("OK")

	// Create feature directory.
	if err := os.MkdirAll(featDir, 0755); err != nil {
		return fmt.Errorf("creating feature directory: %w", err)
	}

	// Write minimal manifest.yaml.
	manifest := fmt.Sprintf(`id: %s
name: %s
description: ""

patches:
  - file: changes.cdiff
    format: cdiff

config: []
`, id, name)

	if err := os.WriteFile(filepath.Join(featDir, "manifest.yaml"), []byte(manifest), 0644); err != nil {
		return fmt.Errorf("writing manifest.yaml: %w", err)
	}

	// Write minimal mappings.yaml.
	mappings := `# Mappings for string replacements in patch files.
# Each entry links a config key to targets where its value is substituted.
#
# - config_key: tenant_noun
#   targets:
#     - from: "Team"
#       to: "{{tenant_noun:title}}"
#     - lines: [5, 10]
#       from: "teams"
#       to: "{{tenant_noun:plural:lower}}"
[]
`

	if err := os.WriteFile(filepath.Join(featDir, "mappings.yaml"), []byte(mappings), 0644); err != nil {
		return fmt.Errorf("writing mappings.yaml: %w", err)
	}

	fmt.Println("")
	fmt.Printf("Created feature %q (%s)\n", name, id)
	fmt.Printf("  Branch:   %s\n", branch)
	fmt.Printf("  Dir:      features/%s/\n", id)
	fmt.Printf("  Manifest: features/%s/manifest.yaml\n", id)
	fmt.Printf("  Mappings: features/%s/mappings.yaml\n", id)

	return nil
}
