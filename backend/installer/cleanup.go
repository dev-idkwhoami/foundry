package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// cleanup removes configured paths from the installed project
// and deletes the temp clone directory.
func cleanup(projectDir, tempClonePath string, cleanupPaths []string, emit func(string)) error {
	for _, rel := range cleanupPaths {
		target := filepath.Join(projectDir, filepath.FromSlash(rel))
		if _, err := os.Stat(target); err == nil {
			emit(fmt.Sprintf("Removing %s", rel))
			if err := os.RemoveAll(target); err != nil {
				return fmt.Errorf("removing %s: %w", rel, err)
			}
		}
	}

	if tempClonePath != "" {
		emit("Resetting temp clone")
		resetCmd := exec.Command("git", "checkout", ".")
		resetCmd.Dir = tempClonePath
		_ = resetCmd.Run()
	}

	return nil
}
