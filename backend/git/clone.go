package git

import (
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"foundry/backend/appdata"
)

// CloneOrPullTemp ensures a cached clone of the repository exists in
// %APPDATA%/Foundry/tmp/<hash>/ (where hash is derived from the repo URL).
// If the directory already exists it runs git pull to update; otherwise it
// clones fresh. Returns the full path to the cached clone.
func CloneOrPullTemp(repoURL string) (string, error) {
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(repoURL)))
	dest := filepath.Join(appdata.TmpPath(), hash)

	if isGitRepo(dest) {
		if err := gitPull(dest); err != nil {
			return "", fmt.Errorf("pull temp clone: %w", err)
		}
		// Reset any leftover working-tree changes from compat checks.
		gitReset(dest)
		return dest, nil
	}

	// Directory might exist but not be a valid repo — remove and re-clone.
	_ = os.RemoveAll(dest)

	if err := runClone(repoURL, dest); err != nil {
		return "", fmt.Errorf("clone to temp: %w", err)
	}

	return dest, nil
}

// CloneToTarget clones the given repository into <targetDir>/<projectName>/.
func CloneToTarget(repoURL, targetDir, projectName string) error {
	dest := filepath.Join(targetDir, projectName)

	if err := runClone(repoURL, dest); err != nil {
		return fmt.Errorf("clone to target: %w", err)
	}

	return nil
}

func runClone(repoURL, dest string) error {
	cmd := exec.Command("git", "clone", repoURL, dest)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, output)
	}

	return nil
}

func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

func gitPull(dir string) error {
	cmd := exec.Command("git", "pull", "--ff-only")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %w\n%s", err, output)
	}
	return nil
}

func gitReset(dir string) {
	cmd := exec.Command("git", "checkout", ".")
	cmd.Dir = dir
	_ = cmd.Run()
}
