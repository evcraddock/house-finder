package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newCommentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "comments <id>",
		Short: "List comments for a property",
		Long:  "List all comments for a property, newest first.",
		Args:  cobra.ExactArgs(1),
		RunE:  runComments,
	}
}

func runComments(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid property ID: %s", args[0])
	}

	repo, database, err := newCommentRepo()
	if err != nil {
		return err
	}
	defer closeDB(database)

	comments, err := repo.ListByPropertyID(id)
	if err != nil {
		return err
	}

	if isJSON() {
		return printJSON(comments)
	}

	fmt.Printf("Comments for property #%d:\n\n", id)
	printCommentList(comments)
	return nil
}
