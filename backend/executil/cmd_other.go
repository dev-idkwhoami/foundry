//go:build !windows

package executil

import "os/exec"

// Command creates a standard exec.Cmd (no-op on non-Windows).
func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
