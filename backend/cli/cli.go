package cli

import (
	"fmt"
	"os"
)

// Known subcommands. Each maps to a run function that receives the
// remaining args (everything after the subcommand name).
var commands = map[string]func(args []string) error{
	"create":   runCreate,
	"diff":     runDiff,
	"publish":  runPublish,
	"validate": runValidate,
	"check":    runCheck,
}

// HandleSubcommand inspects os.Args for a known subcommand. If found,
// it runs the command and returns true (the caller should exit).
// If no subcommand matches, it returns false so the GUI can start.
func HandleSubcommand(args []string) bool {
	if len(args) < 2 {
		return false
	}

	name := args[1]
	fn, ok := commands[name]
	if !ok {
		return false
	}

	if err := fn(args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "foundry %s: %v\n", name, err)
		os.Exit(1)
	}
	return true
}
