package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

func configureUploadCmd(rootCmd *cobra.Command) error {
	// uploadResultsCmd is the "results" sub-command of "uploads".
	uploadResultsCmd := &cobra.Command{
		Use:     "results [flags] --suite-id=<suite> <args>",
		Short:   "Upload test results to Captain",
		Long:    "'captain upload results' will upload test results from various test runners, such as JUnit or RSpec.",
		Example: `captain upload results --suite-id="JUnit" *.xml`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := positionalArgs
			captain, err := cli.GetService(cmd)
			if err != nil {
				return errors.WithStack(err)
			}
			// TODO: Should also support reading from stdin
			_, err = captain.UploadTestResults(cmd.Context(), suiteID, args)
			return errors.WithStack(err)
		},
	}

	uploadResultsCmd.Flags().StringVar(&githubJobName, "github-job-name", "", "the name of the current Github Job")
	if err := uploadResultsCmd.Flags().MarkDeprecated("github-job-name", "the value will be ignored"); err != nil {
		return errors.WithStack(err)
	}

	uploadResultsCmd.Flags().
		StringVar(&githubJobMatrix, "github-job-matrix", "", "the JSON encoded job-matrix from Github")
	if err := uploadResultsCmd.Flags().MarkDeprecated("github-job-matrix", "the value will be ignored"); err != nil {
		return errors.WithStack(err)
	}

	addFrameworkFlags(uploadResultsCmd)
	addGenericProviderFlags(uploadResultsCmd, &cliArgs.GenericProvider)

	// uploadCmd represents the "upload" sub-command itself
	uploadCmd := &cobra.Command{
		Use:        "upload",
		Short:      "Upload a resource to Captain",
		Deprecated: "use 'captain update' instead.",
	}

	uploadCmd.AddCommand(uploadResultsCmd)
	rootCmd.AddCommand(uploadCmd)
	return nil
}
