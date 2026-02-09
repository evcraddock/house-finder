package cli

import (
	"github.com/spf13/cobra"

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

	srv, err := web.NewServer(database)
	if err != nil {
		return err
	}

	return srv.ListenAndServe(port)
}
