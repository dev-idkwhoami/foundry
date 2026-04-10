//go:build windows

package executil

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// Command creates an exec.Cmd that won't spawn a visible console window.
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNoWindow,
	}
	return cmd
}
