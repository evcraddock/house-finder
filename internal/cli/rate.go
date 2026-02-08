package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newRateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rate <id> <1-4>",
		Short: "Rate a property",
		Long:  "Set a rating (1-4) for a property. 4 is best.",
		Args:  cobra.ExactArgs(2),
		RunE:  runRate,
	}
}

func runRate(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid property ID: %s", args[0])
	}

	rating, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid rating: %s (must be 1-4)", args[1])
	}

	if rating < 1 || rating > 4 {
		return fmt.Errorf("rating must be 1-4, got %d", rating)
	}

	repo, database, err := newPropertyRepo()
	if err != nil {
		return err
	}
	defer closeDB(database)

	if err := repo.UpdateRating(id, rating); err != nil {
		return err
	}

	if isJSON() {
		return printJSON(map[string]interface{}{
			"id":     id,
			"rating": rating,
		})
	}

	fmt.Printf("Property #%d rated %s\n", id, formatRating(int64(rating)))
	return nil
}
