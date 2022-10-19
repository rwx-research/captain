package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
)

var (
	testResults string

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Execute a build- or test-suite",
		Long:  descriptionRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.Wrap(captain.RunSuite(cmd.Context(), args, suiteName, testResults))
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

	rootCmd.AddCommand(runCmd)
}
