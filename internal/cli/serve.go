package cli

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/evcraddock/house-finder/internal/auth"
	"github.com/evcraddock/house-finder/internal/mls"
	"github.com/evcraddock/house-finder/internal/web"
)

func newServeCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web UI",
		Long:  "Start an HTTP server for the web UI.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(port)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "port to listen on")

	return cmd
}

func runServe(port int) error {
	database, err := openDB()
	if err != nil {
		return err
	}
	defer closeDB(database)

	authCfg := auth.ConfigFromEnv()

	// Create MLS client if RAPIDAPI_KEY is set (optional — enables POST /api/properties)
	var mlsClient *mls.Client
	if key := os.Getenv("RAPIDAPI_KEY"); key != "" {
		c, cErr := mls.NewClient(key)
		if cErr != nil {
			log.Printf("Warning: MLS client init failed: %v (POST /api/properties disabled)", cErr)
		} else {
			mlsClient = c
		}
	}

	srv, err := web.NewServer(database, authCfg, mlsClient)
	if err != nil {
		return err
	}

	if authCfg.DevMode {
		log.Printf("[DEV] Admin: %s", authCfg.AdminEmail)
		log.Printf("[DEV] Login: http://localhost:%d/login", port)
	}

	return srv.ListenAndServe(port)
}
