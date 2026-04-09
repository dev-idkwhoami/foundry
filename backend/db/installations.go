package db

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Installation represents a tracked Laravel project installation.
type Installation struct {
	ID          int64  `json:"id"`
	PathHash    string `json:"pathHash"`
	ProjectPath string `json:"projectPath"`
	ProjectName string `json:"projectName"`
	Repository  string `json:"repository"`
	SiteName    string `json:"siteName"`
	DbName      string `json:"dbName"`
	InstalledAt string `json:"installedAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// HashPath returns a SHA1 hex digest of the lowercased, cleaned path.
func HashPath(path string) string {
	normalized := strings.ToLower(filepath.Clean(path))
	h := sha1.Sum([]byte(normalized))
	return fmt.Sprintf("%x", h)
}

// InstalledFeature represents a feature installed within a project.
type InstalledFeature struct {
	ID             int64  `json:"id"`
	InstallationID int64  `json:"installationId"`
	FeatureID      string `json:"featureId"`
	FeatureName    string `json:"featureName"`
	ConfigValues   string `json:"configValues"`
	InstalledAt    string `json:"installedAt"`
}

// RecordInstallation upserts an installation keyed on project_path and returns
// the installation ID.
func RecordInstallation(inst Installation) (int64, error) {
	now := time.Now().Format(time.RFC3339)
	pathHash := HashPath(inst.ProjectPath)

	res, err := instance.Exec(`
		INSERT INTO installations (path_hash, project_path, project_name, repository, site_name, db_name, installed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path_hash) DO UPDATE SET
			project_path = excluded.project_path,
			project_name = excluded.project_name,
			repository   = excluded.repository,
			site_name    = excluded.site_name,
			db_name      = excluded.db_name,
			updated_at   = excluded.updated_at
	`, pathHash, inst.ProjectPath, inst.ProjectName, inst.Repository, inst.SiteName, inst.DbName, now, now)
	if err != nil {
		return 0, fmt.Errorf("db: record installation: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("db: record installation last id: %w", err)
	}

	// When ON CONFLICT triggers an UPDATE, LastInsertId may return 0.
	// Fall back to querying by path_hash.
	if id == 0 {
		err = instance.QueryRow(
			`SELECT id FROM installations WHERE path_hash = ?`, pathHash,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("db: record installation lookup: %w", err)
		}
	}

	return id, nil
}

// RecordFeatures replaces all features for the given installation within a
// single transaction.
func RecordFeatures(installationID int64, features []InstalledFeature) error {
	tx, err := instance.Begin()
	if err != nil {
		return fmt.Errorf("db: record features begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM installed_features WHERE installation_id = ?`, installationID); err != nil {
		return fmt.Errorf("db: record features delete: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO installed_features (installation_id, feature_id, feature_name, config_values, installed_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("db: record features prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().Format(time.RFC3339)
	for _, f := range features {
		installedAt := f.InstalledAt
		if installedAt == "" {
			installedAt = now
		}
		configValues := f.ConfigValues
		if configValues == "" {
			configValues = "{}"
		}
		if _, err := stmt.Exec(installationID, f.FeatureID, f.FeatureName, configValues, installedAt); err != nil {
			return fmt.Errorf("db: record feature %q: %w", f.FeatureID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: record features commit: %w", err)
	}

	return nil
}

// GetInstallationByPath returns the installation for the given project path,
// or nil if none exists.
func GetInstallationByPath(projectPath string) (*Installation, error) {
	pathHash := HashPath(projectPath)
	var inst Installation
	err := instance.QueryRow(`
		SELECT id, path_hash, project_path, project_name, repository, site_name, db_name, installed_at, updated_at
		FROM installations WHERE path_hash = ?
	`, pathHash).Scan(
		&inst.ID, &inst.PathHash, &inst.ProjectPath, &inst.ProjectName, &inst.Repository,
		&inst.SiteName, &inst.DbName, &inst.InstalledAt, &inst.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("db: get installation by path: %w", err)
	}
	return &inst, nil
}

// ListInstallations returns all tracked installations.
func ListInstallations() ([]Installation, error) {
	rows, err := instance.Query(`
		SELECT id, path_hash, project_path, project_name, repository, site_name, db_name, installed_at, updated_at
		FROM installations ORDER BY installed_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("db: list installations: %w", err)
	}
	defer rows.Close()

	var list []Installation
	for rows.Next() {
		var inst Installation
		if err := rows.Scan(
			&inst.ID, &inst.PathHash, &inst.ProjectPath, &inst.ProjectName, &inst.Repository,
			&inst.SiteName, &inst.DbName, &inst.InstalledAt, &inst.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("db: list installations scan: %w", err)
		}
		list = append(list, inst)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: list installations rows: %w", err)
	}

	return list, nil
}

// DeleteInstallation removes an installation and its cascading features.
func DeleteInstallation(id int64) error {
	res, err := instance.Exec(`DELETE FROM installations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete installation: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: delete installation rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("db: delete installation: no installation with id %d", id)
	}

	return nil
}
