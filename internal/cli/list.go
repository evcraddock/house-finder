package cli

import (
	"github.com/spf13/cobra"

	"github.com/evcraddock/house-finder/internal/property"
)

func newListCmd() *cobra.Command {
	var minRating int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all properties",
		Long:  "List all tracked properties, optionally filtered by minimum rating.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(minRating)
		},
	}

	cmd.Flags().IntVar(&minRating, "rating", 0, "minimum rating to filter by (1-4)")

	return cmd
}

func runList(minRating int) error {
	repo, database, err := newPropertyRepo()
	if err != nil {
		return err
	}
	defer closeDB(database)

	opts := property.ListOptions{}
	if minRating > 0 {
		opts.MinRating = &minRating
	}

	props, err := repo.List(opts)
	if err != nil {
		return err
	}

	if isJSON() {
		return printJSON(props)
	}

	return printPropertyTable(props)
}
