package main

import (
	"strconv"
	"strings"
)

const (
	GameIDMinecraft = "minecraft"
	GameIDPalworld  = "palworld"
)

type Game struct {
	Installer  string
	Executable string
}

var Games = map[string]Game{
	GameIDMinecraft: {
		Installer:  "java",
		Executable: "server.jar",
	},
	GameIDPalworld: {
		Installer:  "steamcmd",
		Executable: "PalServer.sh",
	},
}

// ReadPlayerCount reads player count from game log output
func ReadPlayerCount(output string) int {
	if output == "" {
		return -1
	}
	parts := strings.Split(output, "There are ")
	if len(parts) < 2 {
		return -1
	}
	numStr := strings.Split(parts[1], " ")[0]
	count, _ := strconv.Atoi(numStr)
	return count
}

func isBlockedConsoleCommand(command string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(command, "/")))
	if normalized == "" || len(normalized) > 512 {
		return true
	}

	for _, blocked := range []string{"stop", "restart", "shutdown", "halt", "kill"} {
		if normalized == blocked || strings.HasPrefix(normalized, blocked+" ") {
			return true
		}
	}
	return false
}
