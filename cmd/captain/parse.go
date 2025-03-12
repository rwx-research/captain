package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
)

func configureParseCmd(rootCmd *cobra.Command, cliArgs *CliArgs) error {
	// parseResultsCmd is the "results" sub-command of "parse".
	parseResultsCmd := &cobra.Command{
		Use:     "results [flags] <test-results-files>",
		Short:   "Parses test-results files into RWX v1 JSON",
		Long:    "'captain parse results' will parse test-results files and output RWX v1 JSON.",
		Example: `captain parse results rspec.json`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: unsafeInitParsingOnly(cliArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			artifacts := cliArgs.RootCliArgs.positionalArgs

			captain, err := cli.GetService(cmd)
			if err != nil {
				return errors.WithStack(err)
			}

			err = captain.Parse(cmd.Context(), artifacts)
			if _, ok := errors.AsConfigurationError(err); !ok {
				cmd.SilenceUsage = true
			}

			return errors.WithStack(err)
		},
	}

	addFrameworkFlags(parseResultsCmd, &cliArgs.frameworkParams)

	// parseCmd represents the "parse" sub-command itself
	parseCmd := &cobra.Command{
		Use:   "parse",
		Short: "Parses a specific resource in captain",
	}

	parseCmd.AddCommand(parseResultsCmd)
	rootCmd.AddCommand(parseCmd)
	return nil
}
