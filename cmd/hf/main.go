// Package main is the entry point for the house-finder CLI.
package main

import (
	"fmt"
	"os"

	"github.com/evcraddock/house-finder/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
