package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
)

var (
	// parseCmd represents the `parse` sub-command itself
	// TODO: consider adding some help text here as well. The command is hidden, but maybe adding a small blurb
	// when running `captain help parse` would be useful.
	parseCmd = &cobra.Command{
		Use:               "parse",
		Hidden:            true,
		PersistentPreRunE: unsafeInitParsingOnly,
	}

	// parseResultsCmd is the 'results' sub-command of 'parse'
	parseResultsCmd = &cobra.Command{
		Use: "results [file]",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.WithStack(captain.Parse(cmd.Context(), args))
		},
	}
)

func init() {
	parseCmd.AddCommand(parseResultsCmd)
	rootCmd.AddCommand(parseCmd)
}
