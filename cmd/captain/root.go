package main

import (
	"os"

	"github.com/spf13/cobra"

	captainCLI "github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/logging"
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
	version         bool

	rootCmd = &cobra.Command{
		Use:           "captain",
		Short:         "Captain makes your builds faster and more reliable",
		Long:          descriptionCaptain,
		SilenceErrors: true, // Errors are manually printed in 'main'
		SilenceUsage:  true, // Disables usage text on error
		RunE: func(cmd *cobra.Command, args []string) error {
			if version {
				logging.NewProductionLogger().Infoln(captainCLI.Version)
				return nil
			}

			return errors.WithStack(cmd.Usage())
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configFilePath, "config-file", "", "The config file for captain")

	_ = addSuiteIDFlagOptionally(rootCmd, &suiteID)

	rootCmd.PersistentFlags().StringVar(&githubJobName, "github-job-name", "", "the name of the current Github Job")
	if err := rootCmd.PersistentFlags().MarkDeprecated("github-job-name", "the value will be ignored"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().StringVar(
		&githubJobMatrix, "github-job-matrix", "", "the JSON encoded job-matrix from Github",
	)
	if err := rootCmd.PersistentFlags().MarkDeprecated("github-job-matrix", "the value will be ignored"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	if err := rootCmd.PersistentFlags().MarkHidden("debug"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "disable TLS for the API")
	if err := rootCmd.PersistentFlags().MarkHidden("insecure"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.Flags().BoolVar(&version, "version", false, "print the Captain CLI version")
	if err := rootCmd.Flags().MarkHidden("version"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true
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

func addSuiteIDFlagOptionally(cmd *cobra.Command, destination *string) string {
	suiteIDFromEnv := os.Getenv("CAPTAIN_SUITE_ID")
	cmd.Flags().StringVar(&suiteID, "suite-id", suiteIDFromEnv,
		"the id of the test suite (required). Also set with environment variable CAPTAIN_SUITE_ID")
	return suiteIDFromEnv
}

// Although `suite-id` is a global flag, we need to re-define it wherever we want to make it required
// This is due to a bug in 'spf13/cobra'. See https://github.com/spf13/cobra/issues/921
func addSuiteIDFlag(cmd *cobra.Command, destination *string) {
	suiteIDFromEnv := addSuiteIDFlagOptionally(cmd, destination)

	// I wonder if there's a more graceful way here?
	if suiteIDFromEnv == "" {
		if err := cmd.MarkFlagRequired("suite-id"); err != nil {
			// smell: global
			initializationErrors = append(initializationErrors, err)
		}
	}
}
