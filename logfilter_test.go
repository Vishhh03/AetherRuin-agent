package main

import (
	"strings"
	"testing"
)

func TestLogFilterDeduplication(t *testing.T) {
	lf := NewLogFilter()

	input := []string{
		"[WARN] Can't keep up!",
		"[WARN] Can't keep up!",
		"[WARN] Can't keep up!",
		"[WARN] Can't keep up!",
		"[WARN] Can't keep up!",
		"[INFO] Server loaded",
	}

	filtered := lf.Filter(input)

	if len(filtered) != 5 {
		t.Fatalf("expected 5 output lines (3 repeats + 1 suppression notice + 1 info), got %d", len(filtered))
	}

	if !strings.Contains(filtered[3], "Suppressed repeated log lines") {
		t.Errorf("expected suppression notice at index 3, got %q", filtered[3])
	}

	if filtered[4] != "[INFO] Server loaded" {
		t.Errorf("expected info line at index 4, got %q", filtered[4])
	}
}

func TestLogFilterTruncation(t *testing.T) {
	lf := NewLogFilter()

	longLine := strings.Repeat("A", 3000)
	filtered := lf.Filter([]string{longLine})

	if len(filtered) != 1 {
		t.Fatalf("expected 1 line, got %d", len(filtered))
	}

	if !strings.HasSuffix(filtered[0], "... (truncated)") {
		t.Errorf("expected line to end with truncation notice")
	}

	if len(filtered[0]) > 2100 {
		t.Errorf("expected truncated line length < 2100, got %d", len(filtered[0]))
	}
}
