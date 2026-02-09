// Package auth provides authentication via magic link email and sessions.
package auth

import "os"

// Config holds authentication configuration.
type Config struct {
	AdminEmail string
	SMTPHost   string
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string
	SMTPFrom   string
	DevMode    bool
	BaseURL    string // e.g. http://localhost:8080
}

// ConfigFromEnv creates a Config from environment variables.
func ConfigFromEnv() Config {
	return Config{
		AdminEmail: os.Getenv("HF_ADMIN_EMAIL"),
		SMTPHost:   os.Getenv("HF_SMTP_HOST"),
		SMTPPort:   envOrDefault("HF_SMTP_PORT", "587"),
		SMTPUser:   os.Getenv("HF_SMTP_USER"),
		SMTPPass:   os.Getenv("HF_SMTP_PASS"),
		SMTPFrom:   os.Getenv("HF_SMTP_FROM"),
		DevMode:    os.Getenv("HF_DEV_MODE") == "true",
		BaseURL:    envOrDefault("HF_BASE_URL", "http://localhost:8080"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
