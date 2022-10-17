// Package main holds the main command line interface for Captain. The package itself is mainly concerned with
// configuring the necessary options before passing control to `internal/cli`, which holds the business logic itself.
package main

import (
	"fmt"
	"os"
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
	// The print call below acts as a last catch-all in case an error is encountered before we could initialize the
	// CLI. It is complimentary to the 'initializationErrors' above.
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
