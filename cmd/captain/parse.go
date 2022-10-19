package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// TODO: consider adding some help text here as well. The command is hidden, but maybe adding a small blurb
//       when running `captain help parse` would be useful.
var parseCmd = &cobra.Command{
	Use:    "parse",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(captain.Parse(cmd.Context(), args))
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
}
