package cli

import (
	"bytes"
	"testing"
)

// executeCommand runs a command with the given args and captures output.
func executeCommand(args ...string) (string, error) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestRootHelp(t *testing.T) {
	_, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalFlags(t *testing.T) {
	root := NewRootCmd()

	formatFlag := root.PersistentFlags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("expected --format flag to exist")
	}
	if formatFlag.DefValue != "text" {
		t.Errorf("expected --format default 'text', got %q", formatFlag.DefValue)
	}

	dbFlag := root.PersistentFlags().Lookup("db")
	if dbFlag == nil {
		t.Fatal("expected --db flag to exist")
	}
}
