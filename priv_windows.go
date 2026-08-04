//go:build windows

package main

import "os/exec"

// Windows builds (dev only) do not support privilege dropping.
const gameSystemUser = "minecraft"

func buildGameCommand(path string, args ...string) (*exec.Cmd, error) {
	return exec.Command(path, args...), nil
}

func chownToGameUser(path string) error {
	return nil
}
