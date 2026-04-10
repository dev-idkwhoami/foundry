package main

import (
	"fmt"
	"os"

	"foundry/backend/cli"
)

func main() {
	// Shift args so the CLI package sees ["foundry-cli", "diff", ...]
	// the same way it would from the GUI binary.
	if !cli.HandleSubcommand(os.Args) {
		fmt.Fprintln(os.Stderr, "Usage: foundry-cli <command> [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  create     Scaffold a new feature (branch + directory)")
		fmt.Fprintln(os.Stderr, "  diff       Generate a .cdiff from git changes")
		fmt.Fprintln(os.Stderr, "  publish    Commit and push feature changes to its branch")
		fmt.Fprintln(os.Stderr, "  validate   Test all features for patch conflicts")
		fmt.Fprintln(os.Stderr, "  check      Test a feature against a set of other features")
		os.Exit(1)
	}
}
