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
