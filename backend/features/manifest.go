package features

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Feature represents a single installable feature parsed from a manifest.yaml.
type Feature struct {
	ID           string        `yaml:"id" json:"id"`
	Name         string        `yaml:"name" json:"name"`
	Description  string        `yaml:"description" json:"description"`
	Requires     []string      `yaml:"requires" json:"requires"`
	Incompatible []string      `yaml:"incompatible" json:"incompatible"`
	Patches      []Patch       `yaml:"patches" json:"patches"`
	Instructions []Instruction `yaml:"instructions" json:"instructions"`
	Config       []ConfigField `yaml:"config" json:"config"`
	Hooks        Hooks         `yaml:"hooks" json:"hooks"`
}

// Patch describes a file patch to apply during feature installation.
type Patch struct {
	File        string `yaml:"file" json:"file"`
	Mode        string `yaml:"mode" json:"mode"`
	Format      string `yaml:"format" json:"format"` // "cdiff" or empty for legacy git apply
	Instruction string `yaml:"instruction" json:"instruction"`
	Diff        string `yaml:"diff" json:"diff"`
}

// Instruction describes a manual step the user must perform after installation.
type Instruction struct {
	Text string `yaml:"text" json:"text"`
	Copy string `yaml:"copy" json:"copy"`
}

// ConfigField represents a user-configurable option exposed by a feature.
type ConfigField struct {
	Key         string         `yaml:"key" json:"key"`
	Label       string         `yaml:"label" json:"label"`
	Type        string         `yaml:"type" json:"type"`
	Default     string         `yaml:"default" json:"default"`
	Placeholder string         `yaml:"placeholder" json:"placeholder"`
	Options     []ConfigOption `yaml:"options" json:"options"`
}

// ConfigOption represents a single option in a select-type config field.
type ConfigOption struct {
	Value string `yaml:"value" json:"value"`
	Label string `yaml:"label" json:"label"`
}

// Hooks holds commands to run at specific points in the installation pipeline.
type Hooks struct {
	PreClone    []string `yaml:"pre-clone" json:"preClone"`
	PostClone   []string `yaml:"post-clone" json:"postClone"`
	PreHerd     []string `yaml:"pre-herd" json:"preHerd"`
	PostHerd    []string `yaml:"post-herd" json:"postHerd"`
	PrePatch    []string `yaml:"pre-patch" json:"prePatch"`
	PostPatch   []string `yaml:"post-patch" json:"postPatch"`
	PreInstall  []string `yaml:"pre-install" json:"preInstall"`
	PostInstall []string `yaml:"post-install" json:"postInstall"`
	PreCleanup  []string `yaml:"pre-cleanup" json:"preCleanup"`
	PostCleanup []string `yaml:"post-cleanup" json:"postCleanup"`
}

// ParseManifest reads a manifest.yaml file at the given path and returns
// the parsed Feature.
func ParseManifest(path string) (*Feature, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}

	var f Feature
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}

	return &f, nil
}
