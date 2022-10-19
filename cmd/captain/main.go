// Package main holds the main command line interface for Captain. The package itself is mainly concerned with
// configuring the necessary options before passing control to `internal/cli`, which holds the business logic itself.
package main

import (
	"fmt"
	"os"

	"github.com/rwx-research/captain-cli/internal/errors"
)

var initializationErrors []error

func main() {
	if initializationErrors != nil {
		// TODO: Additional context for these error messages?
		fmt.Fprintln(os.Stderr, initializationErrors)
		os.Exit(1)
	}

	// Logging is expected to take place in `internal/cli`, as text output is _the_ primary way of communicating
	// to a user on the terminal and is therefore one of our main concerns.
	// This error here is mainly used to communicate any necessary exit Code.
	if err := rootCmd.Execute(); err != nil {
		if e, ok := errors.AsExecutionError(err); ok {
			os.Exit(e.Code)
		}

		// All errors should be wrapped with an error type from our errors package. If this is not the case, this
		// means that the error originates from 'spf13/cobra` or `spf13/viper`. For example, this could be for
		// a missing flag that was marked as required.
		if !errors.IsRWXError(err) {
			fmt.Fprintln(os.Stderr, err)
		}

		os.Exit(1)
	}
}
