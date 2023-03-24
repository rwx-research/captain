package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

var (
	// updateCmd represents the "update" sub-command itself
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "updates a specific resource in captain",
	}

	// updateResultsCmd is the "results" sub-command of "update".
	updateResultsCmd = &cobra.Command{
		Use:     "results [file]",
		Short:   "Updates captain with new test-results",
		Long:    descriptionUpdateResults,
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Should also support reading from stdin
			_, err := captain.UpdateTestResults(cmd.Context(), suiteID, args)
			return errors.WithStack(err)
		},
	}
)

func init() {
	// Although `suite-id` is a global flag, we need to re-define it here in order to mark it as required.
	// This is due to a bug in 'spf13/cobra'. See https://github.com/spf13/cobra/issues/921
	updateResultsCmd.Flags().StringVar(&suiteID, "suite-id", "", "the id of the test suite (required)")
	addFrameworkFlags(updateResultsCmd)

	if err := updateResultsCmd.MarkFlagRequired("suite-id"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	updateCmd.AddCommand(updateResultsCmd)
	rootCmd.AddCommand(updateCmd)
}
