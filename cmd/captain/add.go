package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

func configureAddCmd(rootCmd *cobra.Command, cliArgs *CliArgs) {
	// auxiliaryFlagSet is a secondary (global) flag set that can be used in case we cannot rely on cobra's internal
	// one. This is case for the add / remove commands, which accept arbitrary flags and require disabling cobra's
	// own flag parsing.
	auxiliaryFlagSet := pflag.NewFlagSet("auxiliary", pflag.ContinueOnError)
	auxiliaryFlagSet.Usage = func() {} // Disable secondary "usage" output in cobra

	// Re-define the global flags from `root`
	auxiliaryFlagSet.StringVar(&cliArgs.RootCliArgs.configFilePath, "config-file", "", "the config file for captain")
	auxiliaryFlagSet.BoolVarP(&cliArgs.quiet, "quiet", "q", false, "disables most default output")

	suiteIDFromEnv := os.Getenv("CAPTAIN_SUITE_ID")

	var auxiliarySuiteID string
	auxiliaryFlagSet.StringVar(&auxiliarySuiteID, "suite-id", suiteIDFromEnv,
		"the id of the test suite (required). Also set with environment variable CAPTAIN_SUITE_ID")

	// This will error in case there are unknown flags - i.e. the expected behaviour for add & remove.
	_ = auxiliaryFlagSet.Parse(os.Args)

	// addCmd represents the "add" sub-command itself
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a resource to captain",
	}

	// addFlakeCmd is the "flake" sub-command of "add".
	addFlakeCmd := &cobra.Command{
		Use:   "flake",
		Short: "Mark a test as flaky",
		Long: "'captain add flake' can be used to mark a test as flaky. To select a test, specify the metadata that " +
			"uniquely identifies a single test.",
		Example: `captain add flake --suite-id "example" --file "./test/controller_spec.rb" --description "My test"`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if auxiliarySuiteID != "" {
				cliArgs.RootCliArgs.suiteID = auxiliarySuiteID
			}
			return initCLIService(cliArgs, providers.Validate)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := cliArgs.RootCliArgs.positionalArgs
			captain, err := cli.GetService(cmd)
			if err != nil {
				return errors.WithStack(err)
			}
			return errors.WithStack(captain.AddFlake(cmd.Context(), args))
		},
		DisableFlagParsing: true,
	}
	addCmd.AddCommand(addFlakeCmd)

	// addQuarantineCmd is the "quarantine" sub-command of "add".

	addQuarantineCmd := &cobra.Command{
		Use:   "quarantine",
		Short: "Quarantine a test in Captain",
		Long: "'captain add quarantine' can be used to quarantine a test. To select a test, specify the metadata that " +
			"uniquely identifies a single test.",
		Example: `captain add quarantine --suite-id "example" --file "./test/controller_spec.rb" --description "My test"`,
		PreRunE: initCLIService(cliArgs, providers.Validate),
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := cliArgs.RootCliArgs.positionalArgs
			captain, err := cli.GetService(cmd)
			if err != nil {
				return errors.WithStack(err)
			}
			return errors.WithStack(captain.AddQuarantine(cmd.Context(), args))
		},
		DisableFlagParsing: true,
	}
	addCmd.AddCommand(addQuarantineCmd)
	rootCmd.AddCommand(addCmd)
}
