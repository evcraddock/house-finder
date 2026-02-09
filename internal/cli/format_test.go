package cli

import "testing"

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		name     string
		dollars  int64
		expected string
	}{
		{"zero", 0, "0"},
		{"small", 999, "999"},
		{"thousands", 250000, "250,000"},
		{"millions", 1000000, "1,000,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPrice(tt.dollars)
			if result != tt.expected {
				t.Errorf("formatPrice(%d) = %q, want %q", tt.dollars, result, tt.expected)
			}
		})
	}
}

func TestFormatRating(t *testing.T) {
	tests := []struct {
		name     string
		rating   int64
		expected string
	}{
		{"one", 1, "★☆☆☆"},
		{"two", 2, "★★☆☆"},
		{"three", 3, "★★★☆"},
		{"four", 4, "★★★★"},
		{"clamp low", 0, "★☆☆☆"},
		{"clamp high", 5, "★★★★"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRating(tt.rating)
			if result != tt.expected {
				t.Errorf("formatRating(%d) = %q, want %q", tt.rating, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world!", 8, "hello..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.max)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, result, tt.expected)
			}
		})
	}
}
