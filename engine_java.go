package main

import (
	"bufio"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// JavaEngine implementation for Minecraft and Java-based game servers
type JavaEngine struct {
	*BaseEngine
}

func (j *JavaEngine) Start() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.running {
		return fmt.Errorf("already running")
	}

	args := []string{
		"-Xms" + j.config.MinMemory,
		"-Xmx" + j.config.MaxMemory,
		"-jar", j.config.ServerJar,
		"nogui",
	}

	var err error
	j.cmd, err = buildGameCommand(j.config.JavaPath, args...)
	if err != nil {
		return err
	}
	j.cmd.Dir = getGameWorkDir()

	j.stdin, err = j.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	j.stdout, err = j.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	j.cmd.Stderr = j.cmd.Stdout

	if err := j.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	j.running = true
	j.ready = false
	j.players = 0
	log.Println("Minecraft server (JavaEngine) started")

	go j.monitorOutput()

	go func() {
		j.cmd.Wait()
		j.mu.Lock()
		j.running = false
		j.ready = false
		j.players = 0
		j.mu.Unlock()
		log.Println("Minecraft process exited")
		if j.onExit != nil {
			j.onExit()
		}
	}()

	return nil
}

func (j *JavaEngine) Stop(graceSeconds int) error {
	j.mu.Lock()
	if !j.running {
		j.mu.Unlock()
		return nil
	}
	j.mu.Unlock()

	if j.PlayerCount() > 0 && graceSeconds > 0 {
		j.SendCommand(fmt.Sprintf("say Server stopping in %d seconds", graceSeconds))
		time.Sleep(time.Duration(graceSeconds) * time.Second)
	}

	j.SendCommand("stop")

	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !j.IsRunning() {
				log.Println("Minecraft stopped gracefully")
				return nil
			}
		case <-deadline:
			log.Println("Minecraft stop timeout, killing process")
			if j.cmd != nil && j.cmd.Process != nil {
				j.cmd.Process.Kill()
			}
			j.mu.Lock()
			j.running = false
			j.ready = false
			j.players = 0
			j.mu.Unlock()
			return nil
		}
	}
}

func (j *JavaEngine) SendCommand(command string) (string, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.running || j.stdin == nil {
		return "", fmt.Errorf("server not running")
	}

	_, err := fmt.Fprintln(j.stdin, command)
	if err != nil {
		return "", fmt.Errorf("write command: %w", err)
	}

	return "", nil
}

func (j *JavaEngine) monitorOutput() {
	scanner := bufio.NewScanner(j.stdout)
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
				if len(batch) > 0 && j.onLog != nil {
					j.onLog(batch)
				}
				return
			case <-ticker.C:
				mu.Lock()
				batch := lines
				lines = nil
				mu.Unlock()

				if len(batch) > 0 && j.onLog != nil {
					j.onLog(batch)
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
		if strings.Contains(lowerLine, "done (") && strings.Contains(lowerLine, "for help") && !j.ready {
			j.mu.Lock()
			j.ready = true
			j.mu.Unlock()
			if j.onReady != nil {
				j.onReady()
			}
		}

		if playerCount := ReadPlayerCount(line); playerCount >= 0 {
			j.mu.Lock()
			j.players = playerCount
			j.mu.Unlock()
		}

		if shouldFlush && j.onLog != nil {
			j.onLog(batch)
		}
	}
}
