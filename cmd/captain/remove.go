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
		Short: "removes a resource to captain",
	}

	// removeFlakeCmd is the "flake" sub-command of "remove".
	removeFlakeCmd = &cobra.Command{
		Use:     "flake",
		Short:   "Mark a test as flaky",
		Long:    descriptionRemoveFlake,
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.WithStack(captain.RemoveFlake(cmd.Context(), args))
		},
		DisableFlagParsing: true,
	}

	// removeQuarantineCmd is the "quarantine" sub-command of "remove".
	removeQuarantineCmd = &cobra.Command{
		Use:     "quarantine",
		Short:   "Quarantine a test in Captain",
		Long:    descriptionRemoveQuarantine,
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
