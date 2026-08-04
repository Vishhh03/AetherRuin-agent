package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	ServerID          string `json:"serverId"`
	RegistrationToken string `json:"registrationToken"`
	AgentJWT          string `json:"agentJwt,omitempty"`
	WorkerURL         string `json:"workerUrl"`
	GameID            string `json:"gameId"`
	Installer         string `json:"installer,omitempty"`
	Executable        string `json:"executable,omitempty"`
	JavaPath          string `json:"javaPath,omitempty"`
	ServerJar         string `json:"serverJar,omitempty"`
	MaxMemory         string `json:"maxMemory,omitempty"`
	MinMemory         string `json:"minMemory,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if config.GameID == "" {
		return nil, fmt.Errorf("config validation failed: gameId is required")
	}

	game, ok := Games[config.GameID]
	if !ok {
		return nil, fmt.Errorf("unsupported game_id: %s", config.GameID)
	}

	if config.Installer == "" {
		config.Installer = game.Installer
	}
	if config.Executable == "" {
		config.Executable = game.Executable
	}

	return &config, nil
}

func SaveConfig(path string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
