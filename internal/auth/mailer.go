package auth

import (
	"fmt"

	"github.com/evcraddock/house-finder/internal/email"
)

// Mailer sends magic link emails.
type Mailer struct {
	config Config
}

// NewMailer creates a mailer with the given config.
func NewMailer(config Config) *Mailer {
	return &Mailer{config: config}
}

// SendMagicLink sends a magic link email or logs it in dev mode.
// Returns the magic link URL (useful for dev mode logging by caller).
func (m *Mailer) SendMagicLink(addr, token string) (string, error) {
	link := fmt.Sprintf("%s/auth/verify?token=%s", m.config.BaseURL, token)

	if m.config.DevMode {
		fmt.Printf("[DEV] Magic link for %s: %s\n", addr, link)
		if !m.smtpConfigured() {
			return link, nil
		}
	}

	subject := "House Finder — Login Link"
	body := fmt.Sprintf(
		"Click the link below to log in to House Finder:\n\n%s\n\nThis link expires in 15 minutes and can only be used once.",
		link,
	)

	if err := m.send(addr, subject, body); err != nil {
		return "", err
	}

	return link, nil
}

// SendCLIMagicLink sends a magic link that redirects to /cli/auth/verify.
func (m *Mailer) SendCLIMagicLink(addr, token string) (string, error) {
	link := fmt.Sprintf("%s/cli/auth/verify?token=%s", m.config.BaseURL, token)

	if m.config.DevMode {
		fmt.Printf("[DEV] CLI magic link for %s: %s\n", addr, link)
		if !m.smtpConfigured() {
			return link, nil
		}
	}

	subject := "House Finder — CLI Login Link"
	body := fmt.Sprintf(
		"Click the link below to log in to the House Finder CLI:\n\n%s\n\nThis link expires in 15 minutes and can only be used once.",
		link,
	)

	if err := m.send(addr, subject, body); err != nil {
		return "", err
	}

	return link, nil
}

func (m *Mailer) smtpConfigured() bool {
	return m.config.SMTPHost != "" && m.config.SMTPFrom != ""
}

func (m *Mailer) send(to, subject, body string) error {
	cfg := email.SMTPConfig{
		Host: m.config.SMTPHost,
		Port: m.config.SMTPPort,
		User: m.config.SMTPUser,
		Pass: m.config.SMTPPass,
		From: m.config.SMTPFrom,
	}

	return email.Send(cfg, []string{to}, subject, body)
}
