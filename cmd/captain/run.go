package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
)

var (
	testResults       string
	failOnUploadError bool

	runCmd = &cobra.Command{
		Use:     "run",
		Short:   "Execute a build- or test-suite",
		Long:    descriptionRun,
		PreRunE: initCLIService,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.Wrap(cmd.Usage())
			}

			runConfig := cli.RunConfig{
				Args:              args,
				ArtifactGlob:      testResults,
				FailOnUploadError: failOnUploadError,
				SuiteName:         suiteName,
			}

			return errors.Wrap(captain.RunSuite(cmd.Context(), runConfig))
		},
	}
)

func init() {
	runCmd.Flags().StringVar(
		&testResults,
		"test-results",
		"",
		"a filepath to a test result - supports globs for multiple result files",
	)

	runCmd.Flags().BoolVar(
		&failOnUploadError,
		"fail-on-upload-error",
		false,
		"return a non-zero exit code in case the artifact upload fails",
	)

	rootCmd.AddCommand(runCmd)
}
