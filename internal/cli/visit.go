package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newVisitCmd() *cobra.Command {
	var notes string

	cmd := &cobra.Command{
		Use:   "visit <id> <date> <type>",
		Short: "Record a visit to a property",
		Long: `Record a visit to a property.

Date format: YYYY-MM-DD
Visit types: showing, drive_by, open_house

Examples:
  hf visit 3 2026-02-08 showing
  hf visit 3 2026-02-08 drive_by --notes "nice neighborhood"`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVisit(args, notes)
		},
	}

	cmd.Flags().StringVarP(&notes, "notes", "n", "", "optional notes about the visit")

	return cmd
}

func runVisit(args []string, notes string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid property ID: %s", args[0])
	}

	date := args[1]
	visitType := strings.ToLower(args[2])

	c := newAPIClient()

	v, err := c.AddVisit(id, date, visitType, notes)
	if err != nil {
		return err
	}

	if isJSON() {
		return printJSON(v)
	}

	fmt.Printf("Visit recorded: %s %s (#%d)\n", v.VisitDate, v.VisitType.Label(), v.ID)
	if v.Notes != "" {
		fmt.Printf("  %s\n", v.Notes)
	}
	return nil
}
