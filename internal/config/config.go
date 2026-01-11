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

	// Logging configuration
	LogLevel   string `yaml:"log_level" json:"log_level"`     // Log level: DEBUG, INFO, WARN, ERROR
	LogFile    string `yaml:"log_file" json:"log_file"`       // Path to log file
	LogConsole bool   `yaml:"log_console" json:"log_console"` // Enable console logging
}

// DefaultConfig returns default settings
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	logPath := ""
	if home != "" {
		logPath = filepath.Join(home, ".irontask", "logs", "irontask.log")
	}

	return &Config{
		Editor:        "vim",
		ConfirmDelete: true,
		LogLevel:      getEnv("IRONTASK_LOG_LEVEL", "INFO"),
		LogFile:       getEnv("IRONTASK_LOG_FILE", logPath),
		LogConsole:    getEnv("IRONTASK_LOG_CONSOLE", "false") == "true",
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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

// Save saves config to ~/.irontask/config.yaml
func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".irontask")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
