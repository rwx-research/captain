package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/errors"
)

type rootCliArgs struct {
	configFilePath  string
	debug           bool
	githubJobName   string
	githubJobMatrix string
	insecure        bool
	rootDir         string
	suiteID         string
	positionalArgs  []string
}

func ConfigureRootCmd(rootCmd *cobra.Command, cliArgs *CliArgs) error {
	rootCmd.
		PersistentFlags().StringVar(&cliArgs.RootCliArgs.configFilePath, "config-file", "", "the config file for captain")

	rootCmd.
		PersistentFlags().StringVar(&cliArgs.RootCliArgs.rootDir, "root-dir", "", "the root directory")

	suiteIDFromEnv := os.Getenv("CAPTAIN_SUITE_ID")
	rootCmd.
		PersistentFlags().StringVar(&cliArgs.RootCliArgs.suiteID, "suite-id", suiteIDFromEnv, "the id of the test suite")

	rootCmd.PersistentFlags().BoolVar(&cliArgs.RootCliArgs.debug, "debug", false, "enable debug output")
	if err := rootCmd.PersistentFlags().MarkHidden("debug"); err != nil {
		return errors.WithStack(err)
	}

	rootCmd.PersistentFlags().BoolVar(&cliArgs.RootCliArgs.insecure, "insecure", false, "disable TLS for the API")
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

func bindRootCmdFlags(cfg Config, rootCliArgs rootCliArgs) Config {
	if rootCliArgs.debug {
		cfg.Output.Debug = true
	}

	if rootCliArgs.insecure {
		cfg.Cloud.Insecure = true
	}

	return cfg
}

func extractSuiteIDFromPositionalArgs(rootCliArgs *rootCliArgs, args []string) error {
	rootCliArgs.positionalArgs = args
	if rootCliArgs.suiteID != "" {
		return nil
	}

	if len(rootCliArgs.positionalArgs) == 0 {
		return errors.NewInputError("required flag \"suite-id\" not set")
	}

	rootCliArgs.suiteID = rootCliArgs.positionalArgs[0]
	rootCliArgs.positionalArgs = rootCliArgs.positionalArgs[1:]

	return nil
}
