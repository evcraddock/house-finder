package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/property"
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

// showOutput is the JSON structure for the show command.
type showOutput struct {
	Property *property.Property `json:"property"`
	Comments []*comment.Comment `json:"comments"`
}

func runShow(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid property ID: %s", args[0])
	}

	database, err := openDB()
	if err != nil {
		return err
	}
	defer closeDB(database)

	propRepo := property.NewRepository(database)
	commentRepo := comment.NewRepository(database)

	p, err := propRepo.GetByID(id)
	if err != nil {
		return err
	}

	comments, err := commentRepo.ListByPropertyID(id)
	if err != nil {
		return err
	}

	if isJSON() {
		return printJSON(showOutput{Property: p, Comments: comments})
	}

	printPropertySummary(p)
	fmt.Println()
	if len(comments) > 0 {
		fmt.Printf("Comments (%d):\n", len(comments))
		printCommentList(comments)
	} else {
		fmt.Println("No comments.")
	}

	return nil
}
