package config

import (
	"os"

	"foundry/backend/appdata"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Repository        string   `yaml:"repository" json:"repository"`
	Setup             []string `yaml:"setup" json:"setup"`
	Cleanup           []string `yaml:"cleanup" json:"cleanup"`
	RecentDirectories []string `yaml:"recent_directories" json:"recentDirectories"`
	FluxLicenseKey    string   `yaml:"flux_license_key" json:"fluxLicenseKey"`
	FluxUsername      string   `yaml:"flux_username" json:"fluxUsername"`
	FluxComposerURL   string   `yaml:"flux_composer_url" json:"fluxComposerUrl"`
}

func Load() (*AppConfig, error) {
	data, err := os.ReadFile(appdata.ConfigPath())
	if err != nil {
		return nil, err
	}

	cfg := &AppConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *AppConfig) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(appdata.ConfigPath(), data, 0644)
}

func (c *AppConfig) AddRecentDirectory(dir string) {
	// Remove if already exists
	filtered := make([]string, 0, len(c.RecentDirectories))
	for _, d := range c.RecentDirectories {
		if d != dir {
			filtered = append(filtered, d)
		}
	}

	// Prepend and cap at 5
	c.RecentDirectories = append([]string{dir}, filtered...)
	if len(c.RecentDirectories) > 5 {
		c.RecentDirectories = c.RecentDirectories[:5]
	}
}
