package appdata

import (
	"os"
	"path/filepath"
)

var basePath string

func Init() error {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		appData = filepath.Join(home, "AppData", "Roaming")
	}

	basePath = filepath.Join(appData, "Foundry")

	dirs := []string{
		basePath,
		filepath.Join(basePath, "logs"),
		filepath.Join(basePath, "tmp"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	configPath := filepath.Join(basePath, "config.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := []byte(`repository: "https://github.com/dev-idkwhoami/foundry-starter"

setup:
  - composer install --no-interaction
  - npm install --no-fund --ignore-scripts
  - php artisan key:generate
  - php artisan storage:link
  - php artisan migrate:fresh --seed

cleanup: []

flux_composer_url: "https://composer.fluxui.dev"

recent_directories: []
`)
		if err := os.WriteFile(configPath, defaultConfig, 0644); err != nil {
			return err
		}
	}

	return nil
}

func Path() string {
	return basePath
}

func ConfigPath() string {
	return filepath.Join(basePath, "config.yml")
}

func LogsPath() string {
	return filepath.Join(basePath, "logs")
}

func TmpPath() string {
	return filepath.Join(basePath, "tmp")
}
