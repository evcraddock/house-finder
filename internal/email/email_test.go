package email

import (
	"strings"
	"testing"

	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/property"
)

func ptr[T any](v T) *T { return &v }

func TestFormatEmail(t *testing.T) {
	props := []PropertyWithComments{
		{
			Property: &property.Property{
				Address:    "123 Main St",
				Price:      ptr(int64(250000)),
				Bedrooms:   ptr(float64(3)),
				Bathrooms:  ptr(float64(2)),
				Sqft:       ptr(int64(1500)),
				RealtorURL: "/realestateandhomes-detail/123-Main-St",
			},
			Comments: []*comment.Comment{
				{Text: "Love the backyard"},
				{Text: "Close to schools"},
			},
		},
		{
			Property: &property.Property{
				Address:    "456 Oak Ave",
				Price:      ptr(int64(325000)),
				Bedrooms:   ptr(float64(4)),
				Bathrooms:  ptr(float64(3)),
				Sqft:       ptr(int64(2200)),
				RealtorURL: "https://www.realtor.com/realestateandhomes-detail/456-Oak-Ave",
			},
		},
	}

	body := FormatEmail(props, "http://localhost:8080")

	// Check structure
	if !strings.Contains(body, "2 properties") {
		t.Error("expected property count")
	}
	if !strings.Contains(body, "1. 123 Main St") {
		t.Error("expected first property")
	}
	if !strings.Contains(body, "2. 456 Oak Ave") {
		t.Error("expected second property")
	}

	// Check details
	if !strings.Contains(body, "$250,000") {
		t.Error("expected formatted price")
	}
	if !strings.Contains(body, "3 bed") {
		t.Error("expected bedrooms")
	}
	if !strings.Contains(body, "1,500 sqft") {
		t.Error("expected sqft")
	}

	// Check realtor URL
	if !strings.Contains(body, "https://www.realtor.com/realestateandhomes-detail/123-Main-St") {
		t.Error("expected realtor URL with prefix")
	}

	// Check comments
	if !strings.Contains(body, "Love the backyard") {
		t.Error("expected comment")
	}
	if !strings.Contains(body, "Close to schools") {
		t.Error("expected second comment")
	}

	// Second property should not have notes section
	lines := strings.Split(body, "\n")
	inSecond := false
	for _, line := range lines {
		if strings.Contains(line, "2. 456 Oak Ave") {
			inSecond = true
		}
		if inSecond && strings.Contains(line, "Notes:") {
			t.Error("second property should not have notes")
		}
	}
}

func TestFormatEmailEmpty(t *testing.T) {
	body := FormatEmail(nil, "")
	if !strings.Contains(body, "0 properties") {
		t.Error("expected zero count")
	}
}

func TestSMTPConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  SMTPConfig
		want bool
	}{
		{"fully configured", SMTPConfig{Host: "smtp.example.com", Port: "587", From: "test@example.com"}, true},
		{"missing host", SMTPConfig{From: "test@example.com"}, false},
		{"missing from", SMTPConfig{Host: "smtp.example.com"}, false},
		{"empty", SMTPConfig{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}
