package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored API key",
		Long:  "Removes the stored API key from the config file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout()
		},
	}
}

func runLogout() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.APIKey == "" {
		fmt.Println("Not logged in.")
		return nil
	}

	cfg.APIKey = ""
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("✓ Logged out. API key removed.")
	return nil
}
