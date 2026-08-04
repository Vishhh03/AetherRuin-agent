package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type SystemStats struct {
	CPU     float64 `json:"cpu"`
	RAMMB   int     `json:"ramMb"`
	Players int     `json:"players"`
}

type cpuSnapshot struct {
	user, system, idle int64
	time               time.Time
}

var lastCPU *cpuSnapshot

func ReadStats() SystemStats {
	return SystemStats{
		CPU:   readCPUDelta(),
		RAMMB: readRAMUsage(),
	}
}

func readCPUDelta() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 5 {
		return 0
	}

	user, _ := strconv.ParseInt(fields[1], 10, 64)
	system, _ := strconv.ParseInt(fields[3], 10, 64)
	idle, _ := strconv.ParseInt(fields[4], 10, 64)
	now := time.Now()

	if lastCPU == nil {
		lastCPU = &cpuSnapshot{user, system, idle, now}
		return 0
	}

	dUser := user - lastCPU.user
	dSystem := system - lastCPU.system
	dIdle := idle - lastCPU.idle
	dTotal := dUser + dSystem + dIdle

	lastCPU = &cpuSnapshot{user, system, idle, now}

	if dTotal == 0 {
		return 0
	}

	return float64(dUser+dSystem) / float64(dTotal) * 100
}

func readRAMUsage() int {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}

	var total, available int
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseMemKB(line)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			available = parseMemKB(line)
		}
	}

	used := total - available
	if used < 0 {
		return 0
	}
	return used / 1024
}

func parseMemKB(line string) int {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	val, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return val
}

func (a *Agent) StartStatsLoop(ctx context.Context) {
	// Take initial snapshot
	ReadStats()
	time.Sleep(1 * time.Second)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := ReadStats()
			stats.Players = a.engine.PlayerCount()
			msg := map[string]any{
				"type":    "stats",
				"cpu":     stats.CPU,
				"ramMb":   stats.RAMMB,
				"players": stats.Players,
			}
			a.send(msg)
			if stats.CPU > 80 {
				log.Printf("High CPU: %.1f%%", stats.CPU)
			}
		}
	}
}
