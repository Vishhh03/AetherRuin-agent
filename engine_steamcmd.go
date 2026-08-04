package main

import (
	"bufio"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// SteamCMDEngine implementation for Palworld, Valheim, Rust, ARK, and SteamCMD dedicated servers
type SteamCMDEngine struct {
	*BaseEngine
	executable string
}

func (s *SteamCMDEngine) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("already running")
	}

	var err error
	s.cmd, err = buildGameCommand("/bin/bash", s.executable)
	if err != nil {
		return err
	}
	s.cmd.Dir = getGameWorkDir()
	setProcessGroup(s.cmd)

	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	s.cmd.Stderr = s.cmd.Stdout

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	s.running = true
	s.ready = false
	s.players = 0
	log.Printf("%s server (SteamCMDEngine) started", s.config.GameID)

	go s.monitorOutput()

	go func() {
		s.cmd.Wait()
		s.mu.Lock()
		s.running = false
		s.ready = false
		s.players = 0
		s.mu.Unlock()
		log.Printf("%s process exited", s.config.GameID)
		if s.onExit != nil {
			s.onExit()
		}
	}()

	return nil
}

func (s *SteamCMDEngine) Stop(graceSeconds int) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	_ = killProcessGroup(s.cmd)

	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !s.IsRunning() {
				log.Println("SteamCMD server stopped gracefully")
				return nil
			}
		case <-deadline:
			log.Println("SteamCMD server stop timeout, killing process")
			if s.cmd != nil && s.cmd.Process != nil {
				s.cmd.Process.Kill()
			}
			s.mu.Lock()
			s.running = false
			s.ready = false
			s.players = 0
			s.mu.Unlock()
			return nil
		}
	}
}

func (s *SteamCMDEngine) SendCommand(command string) (string, error) {
	return "", fmt.Errorf("console commands not supported via stdin for this engine")
}

func (s *SteamCMDEngine) monitorOutput() {
	scanner := bufio.NewScanner(s.stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)

	var mu sync.Mutex
	var lines []string

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				mu.Lock()
				batch := lines
				lines = nil
				mu.Unlock()
				if len(batch) > 0 && s.onLog != nil {
					s.onLog(batch)
				}
				return
			case <-ticker.C:
				mu.Lock()
				batch := lines
				lines = nil
				mu.Unlock()

				if len(batch) > 0 && s.onLog != nil {
					s.onLog(batch)
				}
			}
		}
	}()

	for scanner.Scan() {
		line := scanner.Text()

		mu.Lock()
		lines = append(lines, line)
		shouldFlush := len(lines) >= 50
		batch := lines
		if shouldFlush {
			lines = nil
		}
		mu.Unlock()

		lowerLine := strings.ToLower(line)
		if (strings.Contains(lowerLine, "palserver") || strings.Contains(lowerLine, "server created") || strings.Contains(lowerLine, "setting breakpad minidump")) && !s.ready {
			s.mu.Lock()
			s.ready = true
			s.mu.Unlock()
			if s.onReady != nil {
				s.onReady()
			}
		}

		if shouldFlush && s.onLog != nil {
			s.onLog(batch)
		}
	}
}
