package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newCommentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   `comment <id> "text"`,
		Short: "Add a comment to a property",
		Long:  "Add a text comment to a property.",
		Args:  cobra.MinimumNArgs(2),
		RunE:  runComment,
	}
}

func runComment(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid property ID: %s", args[0])
	}

	text := strings.Join(args[1:], " ")
	if text == "" {
		return fmt.Errorf("comment text is required")
	}

	c := newAPIClient()

	comm, err := c.AddComment(id, text)
	if err != nil {
		return err
	}

	if isJSON() {
		return printJSON(comm)
	}

	printCommentSingle(comm)
	return nil
}
