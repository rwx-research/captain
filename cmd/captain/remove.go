package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

var (
	// removeCmd represents the "remove" sub-command itself
	removeCmd = &cobra.Command{
		Use:   "remove",
		Short: "Removes a resource from captain",
	}

	// removeFlakeCmd is the "flake" sub-command of "remove".
	removeFlakeCmd = &cobra.Command{
		Use:   "flake",
		Short: "Mark a test as flaky",
		Long: "'captain remove flake' can be used to remove a specific test for the list of flakes. Effectively, this is " +
			"the inverse of 'captain add flake'.",
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.WithStack(captain.RemoveFlake(cmd.Context(), args))
		},
		DisableFlagParsing: true,
	}

	// removeQuarantineCmd is the "quarantine" sub-command of "remove".
	removeQuarantineCmd = &cobra.Command{
		Use:   "quarantine",
		Short: "Quarantine a test in Captain",
		Long: "'captain remove quarantine' can be used to remove a quarantine from a specific test. Effectively, this is " +
			"the inverse of 'captain add quarantine'.",
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.WithStack(captain.RemoveQuarantine(cmd.Context(), args))
		},
		DisableFlagParsing: true,
	}
)

func init() {
	removeCmd.AddCommand(removeFlakeCmd)
	removeCmd.AddCommand(removeQuarantineCmd)
	rootCmd.AddCommand(removeCmd)
}
