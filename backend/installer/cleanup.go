package installer

import (
	"fmt"
	"os"
	"path/filepath"

	"foundry/backend/executil"
)

// cleanup removes the features directory from the installed project
// and resets the temp clone.
func cleanup(projectDir, tempClonePath string, emit func(string)) error {
	featuresDir := filepath.Join(projectDir, "features")
	if _, err := os.Stat(featuresDir); err == nil {
		emit("Removing features/")
		if err := os.RemoveAll(featuresDir); err != nil {
			return fmt.Errorf("removing features: %w", err)
		}
	}

	if tempClonePath != "" {
		emit("Resetting temp clone")
		resetCmd := executil.Command("git", "checkout", ".")
		resetCmd.Dir = tempClonePath
		_ = resetCmd.Run()
	}

	return nil
}
