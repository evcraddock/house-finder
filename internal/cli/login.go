package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var server string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in and store an API key",
		Long:  "Opens a browser to authenticate and generate an API key for CLI access.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(server)
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "server URL (default: from config or http://localhost:8080)")

	return cmd
}

func runLogin(serverFlag string) error {
	serverURL := serverFlag
	if serverURL == "" {
		serverURL = getServerURL()
	}

	authURL := strings.TrimRight(serverURL, "/") + "/cli/auth"

	fmt.Println("Opening browser for authentication...")
	fmt.Printf("If the browser doesn't open, visit: %s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
	}

	fmt.Print("Paste your API key: ")
	reader := bufio.NewReader(os.Stdin)
	key, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	key = strings.TrimSpace(key)
	if err := validateAPIKey(key); err != nil {
		return err
	}

	// Load existing config to preserve other fields
	cfg, err := loadConfig()
	if err != nil {
		cfg = CLIConfig{}
	}

	cfg.APIKey = key
	if serverFlag != "" {
		cfg.ServerURL = serverFlag
	}

	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("✓ API key saved. You're logged in!")
	return nil
}

// validateAPIKey checks that the key is non-empty and has the expected prefix.
func validateAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("no API key provided")
	}
	if !strings.HasPrefix(key, "hf_") {
		return fmt.Errorf("invalid API key format (should start with hf_)")
	}
	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
