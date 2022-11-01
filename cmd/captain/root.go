package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	captainCLI "github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/logging"
)

var (
	captain       cli.Service
	debug         bool
	githubJobName string
	insecure      bool
	suiteName     string
	version       bool

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

			return errors.Wrap(cmd.Usage())
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&suiteName, "suite-name", "", "the name of the build- or test-suite")

	rootCmd.PersistentFlags().StringVar(&githubJobName, "github-job-name", "", "the name of the current Github Job")
	if err := viper.BindPFlag("ci.github.job", rootCmd.PersistentFlags().Lookup("github-job-name")); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	if err := rootCmd.PersistentFlags().MarkHidden("debug"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}
	if err := viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug")); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "disable TLS for the API")
	if err := rootCmd.PersistentFlags().MarkHidden("insecure"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}
	if err := viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure")); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.Flags().BoolVar(&version, "version", false, "print the Captain CLI version")
	if err := rootCmd.Flags().MarkHidden("version"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
