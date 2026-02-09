package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newVisitsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "visits <id>",
		Short: "List visits for a property",
		Long:  "Show all recorded visits for a property.",
		Args:  cobra.ExactArgs(1),
		RunE:  runVisits,
	}
}

func runVisits(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid property ID: %s", args[0])
	}

	c := newAPIClient()

	visits, err := c.ListVisits(id)
	if err != nil {
		return err
	}

	if isJSON() {
		return printJSON(visits)
	}

	if len(visits) == 0 {
		fmt.Println("No visits recorded.")
		return nil
	}

	printVisits(visits)
	return nil
}
