package main

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type GameEngine interface {
	Start() error
	Stop(graceSeconds int) error
	SendCommand(command string) (string, error)
	IsRunning() bool
	IsReady() bool
	PlayerCount() int
	SetCallbacks(onLog func(lines []string), onReady func(), onExit func())
}

func getGameWorkDir() string {
	if _, err := os.Stat("/opt/game"); err == nil {
		return "/opt/game"
	}
	return "/opt/minecraft"
}

func NewEngine(config *Config) GameEngine {
	base := &BaseEngine{
		config: config,
	}
	workDir := getGameWorkDir()
	execPath := config.Executable
	if execPath == "" {
		if config.GameID == "palworld" {
			execPath = workDir + "/PalServer.sh"
		} else {
			execPath = config.ServerJar
		}
	}
	if !strings.HasPrefix(execPath, "/") {
		execPath = workDir + "/" + execPath
	}

	if config.Installer == "steamcmd" || config.GameID == "palworld" {
		return &SteamCMDEngine{
			BaseEngine: base,
			executable: execPath,
		}
	}
	return &JavaEngine{
		BaseEngine: base,
	}
}

type BaseEngine struct {
	config  *Config
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	running bool
	ready   bool
	players int
	mu      sync.Mutex

	onLog   func(lines []string)
	onReady func()
	onExit  func()
}

func (b *BaseEngine) SetCallbacks(onLog func(lines []string), onReady func(), onExit func()) {
	b.onLog = onLog
	b.onReady = onReady
	b.onExit = onExit
}

func (b *BaseEngine) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

func (b *BaseEngine) IsReady() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ready
}

func (b *BaseEngine) PlayerCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.players
}
