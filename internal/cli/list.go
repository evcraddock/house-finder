package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/evcraddock/house-finder/internal/client"
)

func newListCmd() *cobra.Command {
	var (
		minRating  int
		visited    bool
		notVisited bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all properties",
		Long:  "List all tracked properties, optionally filtered by rating or visit status.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if visited && notVisited {
				return fmt.Errorf("cannot use --visited and --not-visited together")
			}
			opts := client.ListOptions{MinRating: minRating}
			if visited {
				v := true
				opts.Visited = &v
			}
			if notVisited {
				v := false
				opts.Visited = &v
			}
			return runList(opts)
		},
	}

	cmd.Flags().IntVar(&minRating, "rating", 0, "minimum rating to filter by (1-4)")
	cmd.Flags().BoolVar(&visited, "visited", false, "show only visited properties")
	cmd.Flags().BoolVar(&notVisited, "not-visited", false, "show only not-visited properties")

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
