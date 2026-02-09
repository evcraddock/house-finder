package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check connection and auth status",
		Long:  "Tests the connection to the server and checks if the stored API key is valid.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func runStatus() error {
	serverURL := getServerURL()
	apiKey := getAPIKey()

	fmt.Printf("Server:  %s\n", serverURL)

	if apiKey == "" {
		fmt.Println("API Key: not configured")
		fmt.Println("\nRun 'hf login' to authenticate.")
		return nil
	}

	prefix := apiKey
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	fmt.Printf("API Key: %s…\n", prefix)

	// Test the connection with a simple API request
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", serverURL+"/api/properties", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Status:  ✗ cannot reach server (%v)\n", err)
		return nil
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Printf("warning: closing response body: %v\n", cerr)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println("Status:  ✓ connected and authenticated")
	case http.StatusUnauthorized:
		fmt.Println("Status:  ✗ invalid API key")
		fmt.Println("\nRun 'hf login' to re-authenticate.")
	default:
		fmt.Printf("Status:  ✗ unexpected response (%d)\n", resp.StatusCode)
	}

	return nil
}
