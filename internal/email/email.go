// Package email provides email formatting and SMTP sending for house-finder.
package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/property"
)

// SMTPConfig holds SMTP connection settings.
type SMTPConfig struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// IsConfigured returns true if SMTP settings are present.
func (c SMTPConfig) IsConfigured() bool {
	return c.Host != "" && c.From != ""
}

// PropertyWithComments pairs a property with its comments for email formatting.
type PropertyWithComments struct {
	Property *property.Property
	Comments []*comment.Comment
}

// FormatEmail builds a plain-text email body with property details.
func FormatEmail(props []PropertyWithComments, baseURL string) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "Hi,\n\nHere are %d properties I'd like to see:\n\n", len(props))

	for i, pc := range props {
		p := pc.Property

		fmt.Fprintf(&buf, "%d. %s\n", i+1, p.Address)

		var details []string
		if p.Price != nil {
			details = append(details, fmt.Sprintf("$%s", formatWithCommas(*p.Price)))
		}
		if p.Bedrooms != nil {
			details = append(details, fmt.Sprintf("%.0f bed", *p.Bedrooms))
		}
		if p.Bathrooms != nil {
			details = append(details, fmt.Sprintf("%.0f bath", *p.Bathrooms))
		}
		if p.Sqft != nil {
			details = append(details, fmt.Sprintf("%s sqft", formatWithCommas(*p.Sqft)))
		}
		if len(details) > 0 {
			fmt.Fprintf(&buf, "   %s\n", strings.Join(details, " | "))
		}

		if p.RealtorURL != "" {
			url := p.RealtorURL
			if !strings.HasPrefix(url, "http") {
				url = "https://www.realtor.com" + url
			}
			fmt.Fprintf(&buf, "   %s\n", url)
		}

		if len(pc.Comments) > 0 {
			fmt.Fprintf(&buf, "   Notes:\n")
			for _, c := range pc.Comments {
				fmt.Fprintf(&buf, "   - %s\n", c.Text)
			}
		}

		fmt.Fprintln(&buf)
	}

	fmt.Fprintf(&buf, "Thanks!\n")

	return buf.String()
}

// Send sends an email via SMTP.
// Supports both port 465 (implicit TLS) and port 587 (STARTTLS).
func Send(cfg SMTPConfig, to []string, subject, body string) error {
	if !cfg.IsConfigured() {
		return fmt.Errorf("SMTP not configured")
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		cfg.From,
		strings.Join(to, ", "),
		subject,
		body,
	)

	addr := cfg.Host + ":" + cfg.Port

	if cfg.Port == "465" {
		return sendImplicitTLS(cfg, addr, to, msg)
	}
	return sendSTARTTLS(cfg, addr, to, msg)
}

// sendImplicitTLS connects over TLS directly (port 465/SMTPS).
func sendImplicitTLS(cfg SMTPConfig, addr string, to []string, msg string) error {
	tlsCfg := &tls.Config{ServerName: cfg.Host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("TLS dial: %w", err)
	}

	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("creating SMTP client: %w", err)
	}
	defer func() {
		if quitErr := c.Quit(); quitErr != nil {
			err = fmt.Errorf("quit: %w", quitErr)
		}
	}()

	if cfg.User != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err := c.Mail(cfg.From); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt to %s: %w", rcpt, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return nil
}

// sendSTARTTLS connects plain then upgrades to TLS (port 587).
func sendSTARTTLS(cfg SMTPConfig, addr string, to []string, msg string) error {
	var auth smtp.Auth
	if cfg.User != "" {
		auth = smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	}

	if err := smtp.SendMail(addr, auth, cfg.From, to, []byte(msg)); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

func formatWithCommas(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}
