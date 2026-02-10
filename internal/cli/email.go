package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/evcraddock/house-finder/internal/client"
)

func newEmailCmd() *cobra.Command {
	var (
		minRating   int
		visitStatus string
		all         bool
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "email [property IDs...]",
		Short: "Email properties to realtor",
		Long: `Send an email with a formatted list of properties.

By default sends only "want to visit" properties.
Use --status to filter by status, or --all to include everything.
Use --dry-run to preview the email without sending.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := client.EmailRequest{DryRun: dryRun}

			// Parse property IDs from args
			for _, arg := range args {
				id, err := strconv.ParseInt(arg, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid property ID %q: %w", arg, err)
				}
				req.PropertyIDs = append(req.PropertyIDs, id)
			}

			// Only apply filters if no specific IDs given
			if len(req.PropertyIDs) == 0 {
				if minRating > 0 {
					req.MinRating = &minRating
				}
				if all {
					// Don't set visit_status — server returns all
					req.VisitStatus = "all"
				} else if visitStatus != "" {
					req.VisitStatus = visitStatus
				}
				// If no status/all flag, server defaults to want_to_visit
			}

			return runEmail(req)
		},
	}

	cmd.Flags().IntVar(&minRating, "rating", 0, "minimum rating to filter by (1-4)")
	cmd.Flags().StringVar(&visitStatus, "status", "", "filter by visit status (not_visited, want_to_visit, visited)")
	cmd.Flags().BoolVar(&all, "all", false, "include all properties")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview email without sending")

	return cmd
}

func runEmail(req client.EmailRequest) error {
	c := newAPIClient()

	resp, err := c.SendEmail(req)
	if err != nil {
		return err
	}

	if req.DryRun {
		fmt.Printf("To: %s\n", strings.Join(resp.To, ", "))
		fmt.Printf("Subject: %s\n", resp.Subject)
		fmt.Println("---")
		fmt.Print(resp.Body)
		return nil
	}

	fmt.Printf("Email sent to %s\n", strings.Join(resp.To, ", "))
	return nil
}
