package main

import (
	"os"

	"github.com/spf13/cobra"

	captainCLI "github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/internal/cli"
)

var (
	cfg             Config
	captain         cli.Service
	configFilePath  string
	debug           bool
	githubJobName   string
	githubJobMatrix string
	insecure        bool
	suiteID         string

	rootCmd = &cobra.Command{
		Use: "captain",
		Long: "Captain provides client-side utilities related to build- and test-suites. This CLI is a complementary " +
			"component to the main WebUI at https://captain.build.",

		Version: captainCLI.Version,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configFilePath, "config-file", "", "the config file for captain")

	suiteIDFromEnv := os.Getenv("CAPTAIN_SUITE_ID")
	rootCmd.PersistentFlags().StringVar(&suiteID, "suite-id", suiteIDFromEnv, "the id of the test suite")

	if suiteIDFromEnv == "" {
		if err := rootCmd.MarkPersistentFlagRequired("suite-id"); err != nil {
			initializationErrors = append(initializationErrors, err)
		}
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	if err := rootCmd.PersistentFlags().MarkHidden("debug"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "disable TLS for the API")
	if err := rootCmd.PersistentFlags().MarkHidden("insecure"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().BoolVarP(&cliArgs.quiet, "quiet", "q", false, "disables most default output")

	rootCmd.CompletionOptions.DisableDefaultCmd = true   // Disable the `completion` command that's built into cobra
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true}) // Do the same for the `help` command.

	// Change `--version` to output only the version number itself
	rootCmd.SetVersionTemplate(`{{ printf "%s\n" .Version }}`)
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
