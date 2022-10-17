package main

import "github.com/spf13/cobra"

var (
	suiteName string

	// uploadCmd represents the "upload" sub-command itself
	uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload a resource to Captain",
	}

	// uploadResultsCmd is the "results" sub-command of "uploads".
	uploadResultsCmd = &cobra.Command{
		Use:   "results [file]",
		Short: "Upload test results to Captain",
		Long:  descriptionUploadResults,
		Run: func(cmd *cobra.Command, args []string) {
			captain.UploadTestResults(cmd.Context(), suiteName, args)
		},
	}
)

func init() {
	uploadResultsCmd.Flags().StringVar(&suiteName, "suite-name", "", "the suite name (required)")

	// TODO: We've decided to make this optional as well
	if err := uploadResultsCmd.MarkFlagRequired("suite-name"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	uploadCmd.AddCommand(uploadResultsCmd)
	rootCmd.AddCommand(uploadCmd)
}
