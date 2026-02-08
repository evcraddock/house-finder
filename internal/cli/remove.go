package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a property",
		Long:  "Remove a property and all its comments.",
		Args:  cobra.ExactArgs(1),
		RunE:  runRemove,
	}
}

func runRemove(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid property ID: %s", args[0])
	}

	repo, database, err := newPropertyRepo()
	if err != nil {
		return err
	}
	defer closeDB(database)

	if err := repo.Delete(id); err != nil {
		return err
	}

	if isJSON() {
		return printJSON(map[string]interface{}{
			"id":      id,
			"removed": true,
		})
	}

	fmt.Printf("Property #%d removed.\n", id)
	return nil
}
