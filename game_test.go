package main

import "testing"

func TestReadPlayerCount(t *testing.T) {
	tests := []struct {
		name string
		line string
		want int
	}{
		{"empty", "", -1},
		{"unrelated", "[Server thread/INFO]: Done", -1},
		{"zero", "There are 0 of a max of 20 players online:", 0},
		{"some", "There are 3 of a max of 20 players online: alex, sam, lee", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadPlayerCount(tt.line); got != tt.want {
				t.Fatalf("ReadPlayerCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIsBlockedConsoleCommand(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"say hello", false},
		{"/time set day", false},
		{"stop", true},
		{"/stop", true},
		{"restart now", true},
		{"", true},
	}

	for _, tt := range tests {
		if got := isBlockedConsoleCommand(tt.command); got != tt.want {
			t.Fatalf("isBlockedConsoleCommand(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}
