// Package main holds the main command line interface for Captain. The package itself is mainly concerned with
// configuring the necessary options before passing control to `internal/cli`, which holds the business logic itself.
package main

import (
	"fmt"
	"os"

	"github.com/rwx-research/captain-cli/internal/errors"
)

func main() {
	if err := ConfigureRootCmd(rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	configureAddCmd()

	// quarantine
	AddQuarantineFlags(quarantineCmd, &cliArgs)
	rootCmd.AddCommand(quarantineCmd)

	// add and remove
	addFrameworkFlags(parseResultsCmd)
	parseCmd.AddCommand(parseResultsCmd)
	rootCmd.AddCommand(parseCmd)

	removeCmd.AddCommand(removeFlakeCmd)
	removeCmd.AddCommand(removeQuarantineCmd)
	rootCmd.AddCommand(removeCmd)

	if err := configurePartitionCmd(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// run
	if err := AddFlags(runCmd, &cliArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runCmd.SetHelpTemplate(helpTemplate)
	runCmd.SetUsageTemplate(shortUsageTemplate)
	rootCmd.AddCommand(runCmd)

	if err := configureUpdateCmd(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := configureUploadCmd(); err != nil {
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
