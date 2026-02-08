// Package cli defines the cobra command tree for house-finder.
package cli

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/db"
	"github.com/evcraddock/house-finder/internal/mls"
	"github.com/evcraddock/house-finder/internal/property"
)

var (
	flagFormat string
	flagDB     string
)

// NewRootCmd creates the root cobra command with global flags.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "house-finder",
		Short:         "Find and track houses for sale",
		Long:          "A tool to find and track houses for sale. Add properties by address, rate them, leave comments, and browse via CLI or web UI.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagFormat, "format", "text", "output format (text|json)")
	root.PersistentFlags().StringVar(&flagDB, "db", "", "SQLite database path (default: ~/.house-finder/houses.db)")

	root.AddCommand(
		newAddCmd(),
		newListCmd(),
		newShowCmd(),
		newRateCmd(),
		newCommentCmd(),
		newCommentsCmd(),
		newRemoveCmd(),
		newServeCmd(),
		newVersionCmd(),
	)

	return root
}

// openDB opens the SQLite database using the --db flag or default path.
func openDB() (*sql.DB, error) {
	path := flagDB
	if path == "" {
		var err error
		path, err = db.DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	return db.Open(path)
}

// newPropertyRepo opens the DB and returns a property repository.
func newPropertyRepo() (*property.Repository, *sql.DB, error) {
	database, err := openDB()
	if err != nil {
		return nil, nil, fmt.Errorf("opening database: %w", err)
	}
	return property.NewRepository(database), database, nil
}

// newCommentRepo opens the DB and returns a comment repository.
func newCommentRepo() (*comment.Repository, *sql.DB, error) {
	database, err := openDB()
	if err != nil {
		return nil, nil, fmt.Errorf("opening database: %w", err)
	}
	return comment.NewRepository(database), database, nil
}

// newMLSClient creates an MLS client from the RAPIDAPI_KEY env var.
func newMLSClient() (*mls.Client, error) {
	key := os.Getenv("RAPIDAPI_KEY")
	if key == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY environment variable is required")
	}
	return mls.NewClient(key)
}

// isJSON returns true if the --format flag is set to json.
func isJSON() bool {
	return flagFormat == "json"
}

// closeDB closes the database, logging any error to stderr.
func closeDB(database *sql.DB) {
	if err := database.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: closing database: %v\n", err)
	}
}
