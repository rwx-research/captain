package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// config is the internal representation of the configuration file.
type config struct {
	Captain struct {
		Host  string
		Token string
	}
	CI struct {
		Github struct {
			Job string
			Run struct {
				Attempt string
				ID      string
			}
		}
	}
	Insecure bool // Disables TLS for the Captain API
	VCS      struct {
		Github struct {
			Repository string
		}
	}
}

func init() {
	cobra.OnInitialize(func() {
		// TODO: Check if these viper errors are ok to present to end-users
		if err := viper.BindEnv("captain.host", "CAPTAIN_HOST"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("captain.token", "CAPTAIN_TOKEN"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.github.job", "GITHUB_JOB"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.github.run.attempt", "GITHUB_RUN_ATTEMPT"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.github.run.id", "GITHUB_RUN_ID"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("vcs.github.repository", "GITHUB_REPOSITORY"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}
	})
}
