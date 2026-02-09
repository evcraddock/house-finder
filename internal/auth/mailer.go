package auth

import (
	"fmt"
	"net/smtp"
	"strings"
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
func (m *Mailer) SendMagicLink(email, token string) (string, error) {
	link := fmt.Sprintf("%s/auth/verify?token=%s", m.config.BaseURL, token)

	if m.config.DevMode {
		fmt.Printf("[DEV] Magic link for %s: %s\n", email, link)
		return link, nil
	}

	subject := "House Finder — Login Link"
	body := fmt.Sprintf(
		"Click the link below to log in to House Finder:\n\n%s\n\nThis link expires in 15 minutes and can only be used once.",
		link,
	)

	msg := buildEmail(m.config.SMTPFrom, email, subject, body)
	addr := fmt.Sprintf("%s:%s", m.config.SMTPHost, m.config.SMTPPort)
	auth := smtp.PlainAuth("", m.config.SMTPUser, m.config.SMTPPass, m.config.SMTPHost)

	if err := smtp.SendMail(addr, auth, m.config.SMTPFrom, []string{email}, msg); err != nil {
		return "", fmt.Errorf("sending email: %w", err)
	}

	return link, nil
}

// SendCLIMagicLink sends a magic link that redirects to /cli/auth/verify.
func (m *Mailer) SendCLIMagicLink(email, token string) (string, error) {
	link := fmt.Sprintf("%s/cli/auth/verify?token=%s", m.config.BaseURL, token)

	if m.config.DevMode {
		fmt.Printf("[DEV] CLI magic link for %s: %s\n", email, link)
		return link, nil
	}

	subject := "House Finder — CLI Login Link"
	body := fmt.Sprintf(
		"Click the link below to log in to the House Finder CLI:\n\n%s\n\nThis link expires in 15 minutes and can only be used once.",
		link,
	)

	msg := buildEmail(m.config.SMTPFrom, email, subject, body)
	addr := fmt.Sprintf("%s:%s", m.config.SMTPHost, m.config.SMTPPort)
	auth := smtp.PlainAuth("", m.config.SMTPUser, m.config.SMTPPass, m.config.SMTPHost)

	if err := smtp.SendMail(addr, auth, m.config.SMTPFrom, []string{email}, msg); err != nil {
		return "", fmt.Errorf("sending email: %w", err)
	}

	return link, nil
}

func buildEmail(from, to, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", to))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}
