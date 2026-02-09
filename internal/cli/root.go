// Package cli defines the cobra command tree for house-finder.
package cli

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/evcraddock/house-finder/internal/client"
	"github.com/evcraddock/house-finder/internal/db"
)

var (
	flagFormat string
	flagDB     string
)

// NewRootCmd creates the root cobra command with global flags.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "hf",
		Short:         "Find and track houses for sale",
		Long:          "A tool to find and track houses for sale. Add properties by address, rate them, leave comments, and browse via CLI or web UI.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagFormat, "format", "text", "output format (text|json)")
	root.PersistentFlags().StringVar(&flagDB, "db", "", "SQLite database path (default: ~/.config/hf/houses.db)")

	root.AddCommand(
		newAddCmd(),
		newListCmd(),
		newShowCmd(),
		newRateCmd(),
		newCommentCmd(),
		newCommentsCmd(),
		newVisitCmd(),
		newVisitsCmd(),
		newRemoveCmd(),
		newServeCmd(),
		newLoginCmd(),
		newLogoutCmd(),
		newStatusCmd(),
		newVersionCmd(),
	)

	return root
}

// openDB opens the SQLite database using the --db flag or default path.
// Used by the serve command to pass the DB to the web server.
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

// newAPIClient creates an HTTP client for the house-finder API.
func newAPIClient() *client.Client {
	return client.New(getServerURL(), getAPIKey())
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
