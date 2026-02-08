package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/property"
)

// printJSON marshals v as indented JSON and writes it to stdout.
func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// printPropertySummary prints a single property summary in text format.
func printPropertySummary(p *property.Property) {
	fmt.Printf("Property #%d\n", p.ID)
	fmt.Printf("  Address:  %s\n", p.Address)
	fmt.Printf("  URL:      %s\n", p.RealtorURL)
	if p.Price != nil {
		fmt.Printf("  Price:    $%s\n", formatPrice(*p.Price))
	}
	if p.Bedrooms != nil {
		fmt.Printf("  Beds:     %g\n", *p.Bedrooms)
	}
	if p.Bathrooms != nil {
		fmt.Printf("  Baths:    %g\n", *p.Bathrooms)
	}
	if p.Sqft != nil {
		fmt.Printf("  Sqft:     %d\n", *p.Sqft)
	}
	if p.LotSize != nil {
		fmt.Printf("  Lot:      %.2f acres\n", *p.LotSize)
	}
	if p.YearBuilt != nil {
		fmt.Printf("  Built:    %d\n", *p.YearBuilt)
	}
	if p.PropertyType != nil {
		fmt.Printf("  Type:     %s\n", *p.PropertyType)
	}
	if p.Status != nil {
		fmt.Printf("  Status:   %s\n", *p.Status)
	}
	if p.Rating != nil {
		fmt.Printf("  Rating:   %s\n", formatRating(*p.Rating))
	}
}

// printPropertyTable prints a list of properties as a formatted table.
func printPropertyTable(props []*property.Property) error {
	if len(props) == 0 {
		fmt.Println("No properties found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tADDRESS\tPRICE\tBED\tBATH\tSQFT\tRATING"); err != nil {
		return fmt.Errorf("writing table header: %w", err)
	}
	if _, err := fmt.Fprintln(w, "--\t-------\t-----\t---\t----\t----\t------"); err != nil {
		return fmt.Errorf("writing table separator: %w", err)
	}

	for _, p := range props {
		price := "-"
		if p.Price != nil {
			price = "$" + formatPrice(*p.Price)
		}
		beds := "-"
		if p.Bedrooms != nil {
			beds = fmt.Sprintf("%g", *p.Bedrooms)
		}
		baths := "-"
		if p.Bathrooms != nil {
			baths = fmt.Sprintf("%g", *p.Bathrooms)
		}
		sqft := "-"
		if p.Sqft != nil {
			sqft = fmt.Sprintf("%d", *p.Sqft)
		}
		rating := "-"
		if p.Rating != nil {
			rating = formatRating(*p.Rating)
		}

		if _, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			p.ID, truncate(p.Address, 40), price, beds, baths, sqft, rating); err != nil {
			return fmt.Errorf("writing table row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flushing table: %w", err)
	}

	fmt.Printf("\nTotal: %d properties\n", len(props))
	return nil
}

// printCommentList prints comments in text format.
func printCommentList(comments []*comment.Comment) {
	if len(comments) == 0 {
		fmt.Println("No comments.")
		return
	}

	for _, c := range comments {
		fmt.Printf("[%s] #%d\n  %s\n\n",
			c.CreatedAt.Format("2006-01-02 15:04"), c.ID, c.Text)
	}
}

// printCommentSingle prints a single comment in text format.
func printCommentSingle(c *comment.Comment) {
	fmt.Printf("Comment #%d added.\n  %s\n", c.ID, c.Text)
}

// formatPrice formats cents as a dollar string with commas.
func formatPrice(cents int64) string {
	dollars := cents / 100
	s := fmt.Sprintf("%d", dollars)

	// Add commas
	if len(s) <= 3 {
		return s
	}

	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)

	return strings.Join(parts, ",")
}

// formatRating returns a star representation of a rating.
func formatRating(rating int64) string {
	return strings.Repeat("★", int(rating)) + strings.Repeat("☆", 4-int(rating))
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
