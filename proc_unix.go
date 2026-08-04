//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd != nil && cmd.Process != nil {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}
	return nil
}
