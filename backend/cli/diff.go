package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"foundry/backend/executil"
)

// reHunkHeader matches unified diff @@ headers with line numbers.
var reHunkHeader = regexp.MustCompile(`^@@\s.*@@`)

// reDiffFile matches the "diff --git a/<path>" header lines.
var reDiffFile = regexp.MustCompile(`^diff --git a/(.+?) b/`)

// runDiff generates a .cdiff from the current git diff against main.
// It strips line numbers from @@ headers to produce context-only diffs.
//
// Usage: foundry-cli diff [--feature <id>] [--out <path>] [--base <branch>]
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

	// Determine output destination early so we know where to print the summary.
	dest := *outPath
	if dest == "" && *featureID != "" {
		featDir := filepath.Join(cwd, "features", *featureID)
		if err := os.MkdirAll(featDir, 0755); err != nil {
			return fmt.Errorf("creating feature dir: %w", err)
		}
		dest = filepath.Join(featDir, "changes.cdiff")
	}

	// When cdiff goes to stdout, keep the summary on stderr so piping works.
	// When writing to a file, print everything to stdout.
	info := os.Stdout
	if dest == "" {
		info = os.Stderr
	}

	fmt.Fprintf(info, "Diffing against %s...\n", *base)

	// Run git diff excluding the features/ directory.
	cmd := executil.Command("git", "diff", *base, "--", ".", ":(exclude)features/")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("git diff failed: %s", string(exitErr.Stderr))
		}
		return fmt.Errorf("git diff: %w", err)
	}

	if len(out) == 0 {
		fmt.Fprintln(info, "No differences found.")
		return nil
	}

	cdiff := convertToCdiff(string(out))

	// Collect changed files and stats for the summary.
	files, additions, deletions := diffStats(string(out))

	if dest != "" {
		if err := os.WriteFile(dest, []byte(cdiff), 0644); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
	} else {
		fmt.Print(cdiff)
	}

	// Print summary.
	fmt.Fprintln(info, "")
	fmt.Fprintf(info, "  %d file(s) changed, %d insertions(+), %d deletions(-)\n", len(files), additions, deletions)
	for _, f := range files {
		fmt.Fprintf(info, "    %s\n", f)
	}
	if dest != "" {
		fmt.Fprintf(info, "\n  Written to %s\n", dest)
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

// diffStats extracts file names, insertion count, and deletion count
// from a unified diff string.
func diffStats(diff string) (files []string, additions, deletions int) {
	seen := map[string]bool{}

	for _, line := range strings.Split(diff, "\n") {
		if m := reDiffFile.FindStringSubmatch(line); m != nil {
			path := m[1]
			if !seen[path] {
				seen[path] = true
				files = append(files, path)
			}
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}
	return
}
