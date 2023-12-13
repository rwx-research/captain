// Package main holds the main command line interface for Captain. The package itself is mainly concerned with
// configuring the necessary options before passing control to `internal/cli`, which holds the business logic itself.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	captainCLI "github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/internal/errors"
)

func main() {
	rootCmd := &cobra.Command{
		Use: "captain",
		Long: "Captain provides client-side utilities related to build- and test-suites. This CLI is a complementary " +
			"component to the main WebUI at https://cloud.rwx.com/captain",

		Version: captainCLI.Version,
	}
	cliArgs := CliArgs{}

	if err := ConfigureRootCmd(rootCmd, &cliArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// add and remove
	configureAddCmd(rootCmd, &cliArgs)
	configureRemoveCmd(rootCmd, &cliArgs)

	// quarantine
	AddQuarantineFlags(rootCmd, &cliArgs)

	// parse
	if err := configureParseCmd(rootCmd, &cliArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := configurePartitionCmd(rootCmd, &cliArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// run
	runCmd := createRunCmd(&cliArgs)
	if err := AddFlags(runCmd, &cliArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runCmd.SetHelpTemplate(helpTemplate)
	runCmd.SetUsageTemplate(shortUsageTemplate)
	rootCmd.AddCommand(runCmd)

	if err := configureUpdateCmd(rootCmd, &cliArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := configureUploadCmd(rootCmd, &cliArgs); err != nil {
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
