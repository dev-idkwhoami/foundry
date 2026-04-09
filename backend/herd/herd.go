package herd

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
)

// Setup links the project via Herd, creates a PostgreSQL database, and
// configures the .env file with the correct values.
func Setup(projectDir, projectName string) error {
	siteName := toSiteName(projectName)
	dbName := toDBName(projectName)

	if err := linkSite(projectDir, siteName); err != nil {
		return fmt.Errorf("herd link: %w", err)
	}

	if err := createDatabase(dbName); err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	if err := ensureEnvFile(projectDir); err != nil {
		return fmt.Errorf("env file: %w", err)
	}

	if err := configureEnv(projectDir, siteName, dbName); err != nil {
		return fmt.Errorf("configure .env: %w", err)
	}

	return nil
}

// linkSite runs `herd link --secure <siteName>` from the project directory.
func linkSite(projectDir, siteName string) error {
	cmd := exec.Command("herd", "link", "--secure", siteName)
	cmd.Dir = projectDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return nil
}

// createDatabase connects to PostgreSQL and creates the database.
func createDatabase(dbName string) error {
	db, err := sql.Open("postgres", "host=localhost user=root dbname=postgres sslmode=disable")
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer db.Close()

	// CREATE DATABASE cannot be parameterized, so we sanitize the name.
	safe := sanitizeDBName(dbName)
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %q", safe))
	if err != nil {
		// Ignore "already exists" errors.
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return err
	}
	return nil
}

// ensureEnvFile copies .env.example to .env if .env does not exist.
func ensureEnvFile(projectDir string) error {
	envPath := filepath.Join(projectDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		return nil // .env already exists
	}

	examplePath := filepath.Join(projectDir, ".env.example")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return fmt.Errorf("reading .env.example: %w", err)
	}

	if err := os.WriteFile(envPath, data, 0644); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}

	return nil
}

// configureEnv reads the .env file and sets database and APP_URL values.
func configureEnv(projectDir, siteName, dbName string) error {
	envPath := filepath.Join(projectDir, ".env")

	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("reading .env: %w", err)
	}

	content := string(data)

	replacements := map[string]string{
		"DB_CONNECTION": "pgsql",
		"DB_HOST":       "localhost",
		"DB_USERNAME":   "root",
		"DB_PASSWORD":   "",
		"DB_DATABASE":   dbName,
		"APP_URL":       fmt.Sprintf("https://%s.test", siteName),
	}

	for key, val := range replacements {
		content = setEnvValue(content, key, val)
	}

	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}

	return nil
}

// setEnvValue replaces or appends a KEY=VALUE line in .env content.
func setEnvValue(content, key, value string) string {
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `=.*$`)
	replacement := key + "=" + value

	if pattern.MatchString(content) {
		return pattern.ReplaceAllString(content, replacement)
	}
	return content + "\n" + replacement
}

// toSiteName converts a project name to a herd site name (lowercase, hyphens).
func toSiteName(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// toDBName converts a project name to a snake_case database name.
func toDBName(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

// sanitizeDBName removes any characters that are not alphanumeric or underscores.
func sanitizeDBName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return re.ReplaceAllString(name, "")
}
