package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CLIConfig holds CLI configuration persisted to disk.
type CLIConfig struct {
	ServerURL string `yaml:"server_url,omitempty"`
	APIKey    string `yaml:"api_key,omitempty"`
}

// configPath returns the path to the CLI config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".config", "hf", "config.yaml"), nil
}

// loadConfig reads the CLI config from disk.
// Returns a zero-value config if the file doesn't exist.
func loadConfig() (CLIConfig, error) {
	path, err := configPath()
	if err != nil {
		return CLIConfig{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return CLIConfig{}, nil
	}
	if err != nil {
		return CLIConfig{}, fmt.Errorf("reading config: %w", err)
	}

	var cfg CLIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return CLIConfig{}, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// saveConfig writes the CLI config to disk.
func saveConfig(cfg CLIConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// getServerURL returns the server URL from env var, config, or default.
func getServerURL() string {
	if v := os.Getenv("HF_SERVER_URL"); v != "" {
		return v
	}
	cfg, err := loadConfig()
	if err == nil && cfg.ServerURL != "" {
		return cfg.ServerURL
	}
	return "http://localhost:8080"
}

// getAPIKey returns the API key from env var or config.
func getAPIKey() string {
	if v := os.Getenv("HF_API_KEY"); v != "" {
		return v
	}
	cfg, err := loadConfig()
	if err == nil {
		return cfg.APIKey
	}
	return ""
}
