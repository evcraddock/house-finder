package cli

import "testing"

func TestLogoutClearsKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Save a config with an API key
	cfg := CLIConfig{APIKey: "hf_testkey123", ServerURL: "http://myhost:9090"}
	if err := saveConfig(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := runLogout(); err != nil {
		t.Fatalf("logout: %v", err)
	}

	loaded, err := loadConfig()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.APIKey != "" {
		t.Errorf("api_key = %q, want empty after logout", loaded.APIKey)
	}
	// Server URL should be preserved
	if loaded.ServerURL != "http://myhost:9090" {
		t.Errorf("server_url = %q, want preserved after logout", loaded.ServerURL)
	}
}

func TestLogoutWhenNotLoggedIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// No config file — should not error
	if err := runLogout(); err != nil {
		t.Fatalf("logout with no config: %v", err)
	}
}
