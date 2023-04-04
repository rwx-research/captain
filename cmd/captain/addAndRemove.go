package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

func auxiliaryFlagSet(cliArgs *CliArgs) pflag.FlagSet {
	// auxiliaryFlagSet is a secondary (global) flag set that can be used in case we cannot rely on cobra's internal
	// one. This is case for the add / remove commands, which accept arbitrary flags and require disabling cobra's
	// own flag parsing.
	auxiliaryFlagSet := pflag.NewFlagSet("auxiliary", pflag.ContinueOnError)
	auxiliaryFlagSet.Usage = func() {} // Disable secondary "usage" output in cobra

	// Re-define the global flags from `root`
	auxiliaryFlagSet.StringVar(&cliArgs.RootCliArgs.configFilePath, "config-file", "", "the config file for captain")
	auxiliaryFlagSet.BoolVarP(&cliArgs.quiet, "quiet", "q", false, "disables most default output")

	suiteIDFromEnv := os.Getenv("CAPTAIN_SUITE_ID")

	auxiliaryFlagSet.StringVar(&cliArgs.RootCliArgs.suiteID, "suite-id", suiteIDFromEnv,
		"the id of the test suite (required). Also set with environment variable CAPTAIN_SUITE_ID")

	auxiliaryFlagSet.ParseErrorsWhitelist.UnknownFlags = true
	return *auxiliaryFlagSet
}

func noProviderRequired(_ providers.Provider) error { return nil }

func initCLIServiceWithArgs(
	auxiliaryFlagSet pflag.FlagSet, cliArgs *CliArgs,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// This will error in case there are unknown flags - i.e. the expected behaviour for add & remove.
		if err := auxiliaryFlagSet.Parse(args); err != nil {
			return errors.WithDecoration(err)
		}
		return initCLIService(cliArgs, noProviderRequired)(cmd, args)
	}
}

func configureAddCmd(rootCmd *cobra.Command, cliArgs *CliArgs) {
	// addCmd represents the "add" sub-command itself
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a resource to captain",
	}

	auxiliaryFlagSet := auxiliaryFlagSet(cliArgs)

	// addFlakeCmd is the "flake" sub-command of "add".
	addFlakeCmd := &cobra.Command{
		Use:   "flake",
		Short: "Mark a test as flaky",
		Long: "'captain add flake' can be used to mark a test as flaky. To select a test, specify the metadata that " +
			"uniquely identifies a single test.",
		Example: `captain add flake --suite-id "example" --file "./test/controller_spec.rb" --description "My test"`,
		PreRunE: initCLIServiceWithArgs(auxiliaryFlagSet, cliArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			captain, err := cli.GetService(cmd)
			if err != nil {
				return errors.WithStack(err)
			}
			return errors.WithStack(captain.AddFlake(cmd.Context(), args))
		},
		// when using cobra's FParseErrWhitelist, unknown args are only accessible via `os.args` interspersed with cobra args :(
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
		PreRunE: initCLIServiceWithArgs(auxiliaryFlagSet, cliArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
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

func configureRemoveCmd(rootCmd *cobra.Command, cliArgs *CliArgs) {
	// removeCmd represents the "remove" sub-command itself
	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Removes a resource from captain",
	}

	auxiliaryFlagSet := auxiliaryFlagSet(cliArgs)

	// removeFlakeCmd is the "flake" sub-command of "remove".
	removeFlakeCmd := &cobra.Command{
		Use:   "flake",
		Short: "Mark a test as flaky",
		Long: "'captain remove flake' can be used to remove a specific test for the list of flakes. Effectively, this is " +
			"the inverse of 'captain add flake'.",
		PreRunE: initCLIServiceWithArgs(auxiliaryFlagSet, cliArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			captain, err := cli.GetService(cmd)
			if err != nil {
				return errors.WithStack(err)
			}
			return errors.WithStack(captain.RemoveFlake(cmd.Context(), args))
		},
		DisableFlagParsing: true,
	}

	// removeQuarantineCmd is the "quarantine" sub-command of "remove".
	removeQuarantineCmd := &cobra.Command{
		Use:   "quarantine",
		Short: "Quarantine a test in Captain",
		Long: "'captain remove quarantine' can be used to remove a quarantine from a specific test. Effectively, this is " +
			"the inverse of 'captain add quarantine'.",
		PreRunE: initCLIServiceWithArgs(auxiliaryFlagSet, cliArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			captain, err := cli.GetService(cmd)
			if err != nil {
				return errors.WithStack(err)
			}
			return errors.WithStack(captain.RemoveQuarantine(cmd.Context(), args))
		},
		DisableFlagParsing: true,
	}
	removeCmd.AddCommand(removeFlakeCmd)
	removeCmd.AddCommand(removeQuarantineCmd)
	rootCmd.AddCommand(removeCmd)
}
