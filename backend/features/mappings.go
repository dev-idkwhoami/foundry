package features

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Mapping links a config key to one or more file targets where its value
// should be substituted.
type Mapping struct {
	ConfigKey string   `yaml:"config_key" json:"configKey"`
	Targets   []Target `yaml:"targets" json:"targets"`
}

// Target identifies a specific line replacement within a file.
type Target struct {
	Line int    `yaml:"line" json:"line"`
	From string `yaml:"from" json:"from"`
	To   string `yaml:"to" json:"to"`
}

// ParseMappings reads a mappings.yaml file at the given path and returns
// the parsed slice of Mapping entries.
func ParseMappings(path string) ([]Mapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading mappings %s: %w", path, err)
	}

	var mappings []Mapping
	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("parsing mappings %s: %w", path, err)
	}

	return mappings, nil
}
