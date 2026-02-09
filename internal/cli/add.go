package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <address>",
		Short: "Add a property by address",
		Long:  "Look up a property by address using the realtor.com API, fetch its details, and store it.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runAdd,
	}
}

func runAdd(cmd *cobra.Command, args []string) error {
	address := strings.Join(args, " ")

	c := newAPIClient()

	if !isJSON() {
		fmt.Printf("Looking up: %s\n", address)
	}

	p, err := c.AddProperty(address)
	if err != nil {
		return fmt.Errorf("adding property: %w", err)
	}

	if isJSON() {
		return printJSON(p)
	}

	fmt.Println("Property added successfully!")
	printPropertySummary(p)
	return nil
}
