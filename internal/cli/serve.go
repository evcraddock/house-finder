package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
	// Web UI will be implemented in a later task.
	fmt.Printf("Starting web UI on http://localhost:%d\n", port)
	fmt.Println("(not yet implemented — see task #1642)")
	return nil
}
