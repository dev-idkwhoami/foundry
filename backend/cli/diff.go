package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// reHunkHeader matches unified diff @@ headers with line numbers.
var reHunkHeader = regexp.MustCompile(`^@@\s.*@@`)

// runDiff generates a .cdiff from the current git diff against main.
// It strips line numbers from @@ headers to produce context-only diffs.
//
// Usage: foundry diff [--feature <id>] [--out <path>] [--base <branch>]
func runDiff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	featureID := fs.String("feature", "", "Feature ID — auto-places output in features/<id>/")
	outPath := fs.String("out", "", "Output file path (default: stdout)")
	base := fs.String("base", "main", "Base branch to diff against")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Run git diff excluding the features/ directory.
	cmd := exec.Command("git", "diff", *base, "--", ".", ":(exclude)features/")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("git diff failed: %s", string(exitErr.Stderr))
		}
		return fmt.Errorf("git diff: %w", err)
	}

	if len(out) == 0 {
		return fmt.Errorf("no differences found against %s", *base)
	}

	cdiff := convertToCdiff(string(out))

	// Determine output destination.
	dest := *outPath
	if dest == "" && *featureID != "" {
		featDir := filepath.Join(cwd, "features", *featureID)
		if err := os.MkdirAll(featDir, 0755); err != nil {
			return fmt.Errorf("creating feature dir: %w", err)
		}
		dest = filepath.Join(featDir, "changes.cdiff")
	}

	if dest != "" {
		if err := os.WriteFile(dest, []byte(cdiff), 0644); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Written to %s\n", dest)
	} else {
		fmt.Print(cdiff)
	}

	return nil
}

// convertToCdiff strips line numbers from unified diff @@ headers.
func convertToCdiff(gitDiff string) string {
	lines := strings.Split(gitDiff, "\n")
	var result []string

	for _, line := range lines {
		if reHunkHeader.MatchString(line) {
			result = append(result, "@@")
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
