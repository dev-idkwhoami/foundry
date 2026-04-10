package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

// runPublish automates the workflow of publishing a feature's changes
// to a dedicated branch.
//
// Usage: foundry publish --feature <id> [--message <msg>]
//
// Steps:
//  1. Stash current changes
//  2. Checkout (or create) the feature branch: feature/<id>
//  3. Pop the stash
//  4. Stage all changes
//  5. Commit with the provided message
func runPublish(args []string) error {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	featureID := fs.String("feature", "", "Feature ID (required)")
	message := fs.String("message", "", "Commit message")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *featureID == "" {
		return fmt.Errorf("--feature is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	commitMsg := *message
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("feat: update %s feature", *featureID)
	}

	branch := "feature/" + *featureID

	steps := []struct {
		name string
		args []string
	}{
		{"stash changes", []string{"git", "stash", "push", "-m", "foundry-publish-" + *featureID}},
		{"switch to branch", []string{"git", "checkout", "-B", branch}},
		{"pop stash", []string{"git", "stash", "pop"}},
		{"stage changes", []string{"git", "add", "-A"}},
		{"commit", []string{"git", "commit", "-m", commitMsg}},
	}

	for _, step := range steps {
		fmt.Fprintf(os.Stderr, "  %s...\n", step.name)
		cmd := exec.Command(step.args[0], step.args[1:]...)
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	fmt.Fprintf(os.Stderr, "\nPublished to branch %s\n", branch)
	return nil
}
