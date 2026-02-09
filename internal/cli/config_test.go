package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSaveAndLoad(t *testing.T) {
	// Use a temp dir as home
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := CLIConfig{
		ServerURL: "http://myhost:9090",
		APIKey:    "hf_testapikey123",
	}

	if err := saveConfig(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmp, ".config", "hf", "config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not found: %v", err)
	}

	loaded, err := loadConfig()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ServerURL != cfg.ServerURL {
		t.Errorf("server_url = %q, want %q", loaded.ServerURL, cfg.ServerURL)
	}
	if loaded.APIKey != cfg.APIKey {
		t.Errorf("api_key = %q, want %q", loaded.APIKey, cfg.APIKey)
	}
}

func TestConfigLoadMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if cfg.ServerURL != "" || cfg.APIKey != "" {
		t.Error("expected zero-value config for missing file")
	}
}

func TestGetServerURLFromEnv(t *testing.T) {
	t.Setenv("HF_SERVER_URL", "http://custom:1234")
	t.Setenv("HOME", t.TempDir())

	url := getServerURL()
	if url != "http://custom:1234" {
		t.Errorf("url = %q, want %q", url, "http://custom:1234")
	}
}

func TestGetServerURLDefault(t *testing.T) {
	t.Setenv("HF_SERVER_URL", "")
	t.Setenv("HOME", t.TempDir())

	url := getServerURL()
	if url != "http://localhost:8080" {
		t.Errorf("url = %q, want %q", url, "http://localhost:8080")
	}
}

func TestGetAPIKeyFromEnv(t *testing.T) {
	t.Setenv("HF_API_KEY", "hf_envkey")
	t.Setenv("HOME", t.TempDir())

	key := getAPIKey()
	if key != "hf_envkey" {
		t.Errorf("key = %q, want %q", key, "hf_envkey")
	}
}

func TestGetAPIKeyFromConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("HF_API_KEY", "")

	cfg := CLIConfig{APIKey: "hf_configkey"}
	if err := saveConfig(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	key := getAPIKey()
	if key != "hf_configkey" {
		t.Errorf("key = %q, want %q", key, "hf_configkey")
	}
}

func TestGetAPIKeyEmpty(t *testing.T) {
	t.Setenv("HF_API_KEY", "")
	t.Setenv("HOME", t.TempDir())

	key := getAPIKey()
	if key != "" {
		t.Errorf("key = %q, want empty", key)
	}
}
