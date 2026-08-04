package main

import (
	"strings"
	"testing"
)

func TestNewEngine(t *testing.T) {
	mcConfig := &Config{GameID: "minecraft", ServerJar: "server.jar"}
	mcEngine := NewEngine(mcConfig)
	if _, ok := mcEngine.(*JavaEngine); !ok {
		t.Fatalf("expected JavaEngine for minecraft, got %T", mcEngine)
	}

	palConfig := &Config{GameID: "palworld", Installer: "steamcmd", Executable: "PalServer.sh"}
	palEngine := NewEngine(palConfig)
	steamEngine, ok := palEngine.(*SteamCMDEngine)
	if !ok {
		t.Fatalf("expected SteamCMDEngine for palworld, got %T", palEngine)
	}

	if !strings.HasSuffix(steamEngine.executable, "PalServer.sh") {
		t.Fatalf("expected executable to end with PalServer.sh, got %q", steamEngine.executable)
	}
}

func TestEngineReadinessCheck(t *testing.T) {
	palEngine := &SteamCMDEngine{
		BaseEngine: &BaseEngine{config: &Config{GameID: "palworld"}},
	}
	if palEngine.IsReady() {
		t.Fatalf("expected initial readiness to be false")
	}
	if palEngine.IsRunning() {
		t.Fatalf("expected initial running state to be false")
	}
	if palEngine.PlayerCount() != 0 {
		t.Fatalf("expected initial player count to be 0")
	}
}

func TestSteamCMDSendCommandUnsupported(t *testing.T) {
	palEngine := &SteamCMDEngine{
		BaseEngine: &BaseEngine{config: &Config{GameID: "palworld"}},
	}
	_, err := palEngine.SendCommand("test")
	if err == nil {
		t.Fatalf("expected error sending stdin command to SteamCMDEngine, got nil")
	}
}

func TestEngineSetCallbacks(t *testing.T) {
	base := &BaseEngine{config: &Config{GameID: "minecraft"}}
	var logCalled, readyCalled, exitCalled bool

	base.SetCallbacks(
		func(lines []string) { logCalled = true },
		func() { readyCalled = true },
		func() { exitCalled = true },
	)

	if base.onLog == nil || base.onReady == nil || base.onExit == nil {
		t.Fatalf("callbacks were not set properly on BaseEngine")
	}

	base.onLog([]string{"test"})
	base.onReady()
	base.onExit()

	if !logCalled || !readyCalled || !exitCalled {
		t.Fatalf("callbacks were not triggered as expected")
	}
}
