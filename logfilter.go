package main

import (
	"fmt"
	"strings"
	"sync"
)

type LogFilter struct {
	mu          sync.Mutex
	lastLine    string
	repeatCount int
	maxRepeat   int
	maxLineLen  int
}

func NewLogFilter() *LogFilter {
	return &LogFilter{
		maxRepeat:  3,
		maxLineLen: 2048,
	}
}

func (lf *LogFilter) Filter(lines []string) []string {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	var result []string
	for _, line := range lines {
		// Truncate ultra-long lines
		if len(line) > lf.maxLineLen {
			line = line[:lf.maxLineLen] + " ... (truncated)"
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Deduplicate identical repeated lines
		if trimmed == lf.lastLine {
			lf.repeatCount++
			if lf.repeatCount <= lf.maxRepeat {
				result = append(result, line)
			} else if lf.repeatCount == lf.maxRepeat+1 {
				result = append(result, fmt.Sprintf("[AetherRuin Agent] Suppressed repeated log lines: %s", trimmed))
			}
			continue
		}

		// New line encountered
		lf.lastLine = trimmed
		lf.repeatCount = 1
		result = append(result, line)
	}

	return result
}
