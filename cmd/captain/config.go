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
			Job struct {
				Name   string
				Matrix string
			}
			Run struct {
				Attempt         string
				ID              string
				EventName       string
				EventPath       string
				ExecutingActor  string
				TriggeringActor string
			}
		}
	}
	Debug    bool
	Insecure bool // Disables TLS for the Captain API
	VCS      struct {
		Github struct {
			Repository string
			RefName    string
			HeadRef    string
			CommitSha  string
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

		if err := viper.BindEnv("captain.token", "RWX_ACCESS_TOKEN"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("debug", "DEBUG"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.github.job.name", "GITHUB_JOB"); err != nil {
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

		if err := viper.BindEnv("ci.github.run.eventName", "GITHUB_EVENT_NAME"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.github.run.eventPath", "GITHUB_EVENT_PATH"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.github.run.executingActor", "GITHUB_ACTOR"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.github.run.triggeringActor", "GITHUB_TRIGGERING_ACTOR"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("vcs.github.repository", "GITHUB_REPOSITORY"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("vcs.github.refName", "GITHUB_REF_NAME"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("vcs.github.headRef", "GITHUB_HEAD_REF"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("vcs.github.commitSha", "GITHUB_SHA"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}
	})
}
