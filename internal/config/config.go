package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds user preferences
type Config struct {
	Editor        string `yaml:"editor" json:"editor"`                 // Default editor command
	ConfirmDelete bool   `yaml:"confirm_delete" json:"confirm_delete"` // Require confirmation for delete
}

// DefaultConfig returns default settings
func DefaultConfig() *Config {
	return &Config{
		Editor:        "vim",
		ConfirmDelete: true,
	}
}

// Load loads config from ~/.irontask/config.yaml
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".irontask", "config.yaml")

	// Check if exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return defaults if no config
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
