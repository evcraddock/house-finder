package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show property details",
		Long:  "Show full details for a property, including all comments.",
		Args:  cobra.ExactArgs(1),
		RunE:  runShow,
	}
}

func runShow(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid property ID: %s", args[0])
	}

	c := newAPIClient()

	resp, err := c.GetProperty(id)
	if err != nil {
		return err
	}

	if isJSON() {
		return printJSON(resp)
	}

	printPropertySummary(resp.Property)
	fmt.Println()
	if len(resp.Visits) > 0 {
		fmt.Printf("Visits (%d):\n", len(resp.Visits))
		printVisits(resp.Visits)
	}
	if len(resp.Comments) > 0 {
		fmt.Printf("Comments (%d):\n", len(resp.Comments))
		printCommentList(resp.Comments)
	} else {
		fmt.Println("No comments.")
	}

	return nil
}
