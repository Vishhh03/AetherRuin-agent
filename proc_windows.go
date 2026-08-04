//go:build windows

package main

import (
	"os"
	"os/exec"
)

func setProcessGroup(cmd *exec.Cmd) {
	// No-op on Windows
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Signal(os.Interrupt)
	}
	return nil
}
