package cli

import (
	"github.com/spf13/cobra"

	"github.com/evcraddock/house-finder/internal/client"
)

func newListCmd() *cobra.Command {
	var (
		minRating int
		status    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all properties",
		Long:  "List all tracked properties, optionally filtered by rating or visit status.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := client.ListOptions{MinRating: minRating, Status: status}
			return runList(opts)
		},
	}

	cmd.Flags().IntVar(&minRating, "rating", 0, "minimum rating to filter by (1-4)")
	cmd.Flags().StringVar(&status, "status", "", "filter by status (not-visited, scheduled, visited)")

	return cmd
}

func runList(opts client.ListOptions) error {
	c := newAPIClient()

	props, err := c.ListProperties(opts)
	if err != nil {
		return err
	}

	if isJSON() {
		return printJSON(props)
	}

	return printPropertyTable(props)
}
