package main

import "testing"

func TestExample(t *testing.T) {
	result := 1 + 1
	expected := 2

	if result != expected {
		t.Errorf("got %d, want %d", result, expected)
	}
}
