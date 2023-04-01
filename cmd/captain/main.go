// Package main holds the main command line interface for Captain. The package itself is mainly concerned with
// configuring the necessary options before passing control to `internal/cli`, which holds the business logic itself.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	captainCLI "github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

func main() {
	rootCmd := &cobra.Command{
		Use: "captain",
		Long: "Captain provides client-side utilities related to build- and test-suites. This CLI is a complementary " +
			"component to the main WebUI at https://captain.build.",

		Version: captainCLI.Version,
	}

	if err := ConfigureRootCmd(rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	configureAddCmd(rootCmd)

	// quarantine
	AddQuarantineFlags(rootCmd, &cliArgs)

	// add and remove
	// parseCmd represents the `parse` sub-command itself
	// TODO: consider adding some help text here as well. The command is hidden, but maybe adding a small blurb
	// when running `captain help parse` would be useful.
	parseCmd := &cobra.Command{
		Use:               "parse",
		Hidden:            true,
		PersistentPreRunE: unsafeInitParsingOnly,
	}

	// parseResultsCmd is the 'results' sub-command of 'parse'
	parseResultsCmd := &cobra.Command{
		Use: "results [flags] <args>",
		RunE: func(cmd *cobra.Command, args []string) error {
			// note: fetching from the parent command because this command doesn't run PreRunE
			captain, err := cli.GetService(parseCmd)
			if err != nil {
				return errors.WithStack(err)
			}
			return errors.WithStack(captain.Parse(cmd.Context(), args))
		},
	}
	addFrameworkFlags(parseResultsCmd)
	parseCmd.AddCommand(parseResultsCmd)
	rootCmd.AddCommand(parseCmd)

	// removeCmd represents the "remove" sub-command itself
	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Removes a resource from captain",
	}

	// removeFlakeCmd is the "flake" sub-command of "remove".
	removeFlakeCmd := &cobra.Command{
		Use:   "flake",
		Short: "Mark a test as flaky",
		Long: "'captain remove flake' can be used to remove a specific test for the list of flakes. Effectively, this is " +
			"the inverse of 'captain add flake'.",
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := positionalArgs
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
		PreRunE: initCLIService(providers.Validate),
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := positionalArgs
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

	if err := configurePartitionCmd(rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// run
	runCmd := createRunCmd()
	if err := AddFlags(runCmd, &cliArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runCmd.SetHelpTemplate(helpTemplate)
	runCmd.SetUsageTemplate(shortUsageTemplate)
	rootCmd.AddCommand(runCmd)

	if err := configureUpdateCmd(rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := configureUploadCmd(rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Logging is expected to take place in `internal/cli`, as text output is the primary way of communicating
	// to a user on the terminal and is therefore one of our main concerns.
	// This error here is mainly used to communicate any necessary exit Code.
	if err := rootCmd.Execute(); err != nil {
		if e, ok := errors.AsExecutionError(err); ok {
			os.Exit(e.Code)
		}
		os.Exit(1)
	}
}
