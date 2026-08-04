//go:build !windows

package main

import (
	"errors"
	"os/exec"
)

// gameSystemUser is the unprivileged account the game server processes run as.
// The agent itself still runs as root (needed for self-update and systemctl),
// but the game is never started with root privileges. The account is created
// by the cloud-init userdata (apps/workers/queue/src/userdata.ts).
const gameSystemUser = "minecraft"

// buildGameCommand returns an exec.Cmd that runs the given program as the
// unprivileged game user via setpriv(1). It fails closed: if setpriv is not
// available (or the game user does not exist), the game refuses to start
// rather than running as root.
func buildGameCommand(path string, args ...string) (*exec.Cmd, error) {
	setpriv, err := exec.LookPath("setpriv")
	if err != nil {
		return nil, errors.New("setpriv(1) not found: refusing to run game server as root")
	}

	full := []string{
		"--reuid", gameSystemUser,
		"--regid", gameSystemUser,
		"--init-groups",
		"--",
		path,
	}
	full = append(full, args...)

	return exec.Command(setpriv, full...), nil
}

// chownToGameUser reassigns a file (e.g. server.properties written by the
// root-run agent) to the unprivileged game user so the game can modify it.
func chownToGameUser(path string) error {
	chown, err := exec.LookPath("chown")
	if err != nil {
		return err
	}
	cmd := exec.Command(chown, gameSystemUser+":"+gameSystemUser, path)
	return cmd.Run()
}
