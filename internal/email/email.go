// Package email provides email formatting and SMTP sending for house-finder.
package email

import (
	"bytes"
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
