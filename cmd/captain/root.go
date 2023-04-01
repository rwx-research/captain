package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// TODO figure out how to handle shared state
var (
	configFilePath  string
	debug           bool
	githubJobName   string
	githubJobMatrix string
	insecure        bool
	suiteID         string
	positionalArgs  []string
)

func ConfigureRootCmd(rootCmd *cobra.Command) error {
	rootCmd.PersistentFlags().StringVar(&configFilePath, "config-file", "", "the config file for captain")

	suiteIDFromEnv := os.Getenv("CAPTAIN_SUITE_ID")
	rootCmd.PersistentFlags().StringVar(&suiteID, "suite-id", suiteIDFromEnv, "the id of the test suite")

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	if err := rootCmd.PersistentFlags().MarkHidden("debug"); err != nil {
		return errors.WithStack(err)
	}

	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "disable TLS for the API")
	if err := rootCmd.PersistentFlags().MarkHidden("insecure"); err != nil {
		return errors.WithStack(err)
	}

	rootCmd.PersistentFlags().BoolVarP(&cliArgs.quiet, "quiet", "q", false, "disables most default output")

	rootCmd.CompletionOptions.DisableDefaultCmd = true   // Disable the `completion` command that's built into cobra
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true}) // Do the same for the `help` command.

	// Change `--version` to output only the version number itself
	rootCmd.SetVersionTemplate(`{{ printf "%s\n" .Version }}`)
	return nil
}

func bindRootCmdFlags(cfg Config) Config {
	if debug {
		cfg.Output.Debug = true
	}

	if insecure {
		cfg.Cloud.Insecure = true
	}

	return cfg
}

func extractSuiteIDFromPositionalArgs(cmd *cobra.Command) error {
	positionalArgs = cmd.Flags().Args()

	if suiteID != "" {
		return nil
	}

	if len(positionalArgs) == 0 {
		return errors.NewInputError("required flag \"suite-id\" not set")
	}

	suiteID = positionalArgs[0]
	positionalArgs = positionalArgs[1:]

	return nil
}
