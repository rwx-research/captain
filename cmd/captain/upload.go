package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
)

var (
	// uploadCmd represents the "upload" sub-command itself
	uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload a resource to Captain",
	}

	// uploadResultsCmd is the "results" sub-command of "uploads".
	uploadResultsCmd = &cobra.Command{
		Use:     "results [file]",
		Short:   "Upload test results to Captain",
		Long:    descriptionUploadResults,
		PreRunE: initCLIService,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Should also support reading from stdin
			_, err := captain.UploadTestResults(cmd.Context(), suiteID, args)
			return errors.WithStack(err)
		},
	}
)

func init() {
	// Although `suite-id` is a global flag, we need to re-define it here in order to mark it as required.
	// This is due to a bug in 'spf13/cobra'. See https://github.com/spf13/cobra/issues/921
	uploadResultsCmd.Flags().StringVar(&suiteID, "suite-id", "", "the id of the test suite (required)")
	addFrameworkFlags(uploadResultsCmd)

	if err := uploadResultsCmd.MarkFlagRequired("suite-id"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	uploadCmd.AddCommand(uploadResultsCmd)
	rootCmd.AddCommand(uploadCmd)
}
