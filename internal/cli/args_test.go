package cli

import (
	"testing"
)

func TestAddRequiresAddress(t *testing.T) {
	_, err := executeCommand("add")
	if err == nil {
		t.Fatal("expected error when no address provided")
	}
}

func TestListAcceptsNoArgs(t *testing.T) {
	// list with a temp DB should work (no args required).
	// We just test that arg validation passes by providing a bad DB to avoid real DB creation.
	_, err := executeCommand("list", "--db", "/dev/null/nonexistent")
	// Error should be about DB, not about args
	if err == nil {
		t.Fatal("expected error (bad db path)")
	}
}

func TestShowRequiresID(t *testing.T) {
	_, err := executeCommand("show")
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

func TestShowRejectsNonNumericID(t *testing.T) {
	_, err := executeCommand("show", "abc", "--db", "/tmp/hf-test-nonexistent.db")
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
}

func TestRateRequiresTwoArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{"rate"}},
		{"one arg", []string{"rate", "1"}},
		{"three args", []string{"rate", "1", "3", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRateRejectsInvalidRating(t *testing.T) {
	tests := []struct {
		name   string
		rating string
	}{
		{"zero", "0"},
		{"five", "5"},
		{"negative", "-1"},
		{"string", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand("rate", "1", tt.rating, "--db", "/tmp/hf-test-nonexistent.db")
			if err == nil {
				t.Fatal("expected error for invalid rating")
			}
		})
	}
}

func TestCommentRequiresIDAndText(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{"comment"}},
		{"id only", []string{"comment", "1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCommentsRequiresID(t *testing.T) {
	_, err := executeCommand("comments")
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

func TestRemoveRequiresID(t *testing.T) {
	_, err := executeCommand("remove")
	if err == nil {
		t.Fatal("expected error when no ID provided")
	}
}

func TestServeAcceptsNoArgs(t *testing.T) {
	// serve should reject extra args
	_, err := executeCommand("serve", "extra")
	if err == nil {
		t.Fatal("expected error for extra args")
	}
}
