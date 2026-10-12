// Package cmd wires the chores command line interface.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute runs the CLI, exiting non-zero on error.
func Execute() {
	root := &cobra.Command{
		Use:           "chores",
		Short:         "Render a weekly chore chart PDF from a chore list",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(renderCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
