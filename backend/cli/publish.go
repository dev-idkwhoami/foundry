package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"foundry/backend/executil"
)

// runPublish automates the workflow of publishing a feature's changes
// to a dedicated branch.
//
// Usage: foundry-cli publish --feature <id> [--message <msg>]
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
		label string
		args  []string
	}{
		{"Stashing changes", []string{"git", "stash", "push", "-m", "foundry-publish-" + *featureID}},
		{"Switching to branch " + branch, []string{"git", "checkout", "-B", branch}},
		{"Restoring stash", []string{"git", "stash", "pop"}},
		{"Staging changes", []string{"git", "add", "-A"}},
		{"Committing", []string{"git", "commit", "-m", commitMsg}},
	}

	for _, step := range steps {
		fmt.Printf("  %s... ", step.label)
		cmd := executil.Command(step.args[0], step.args[1:]...)
		cmd.Dir = cwd
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("%s: %s", strings.ToLower(step.label), strings.TrimSpace(string(out)))
		}
		fmt.Println("OK")
	}

	fmt.Println("")
	fmt.Printf("Published to branch %s\n", branch)
	return nil
}
