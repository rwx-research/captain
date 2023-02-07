package main

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

// config is the internal representation of the configuration file.
type config struct {
	Captain struct {
		Host  string
		Token string
	}
	CI struct {
		Github struct {
			Detected bool
			Job      struct {
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
		Buildkite struct {
			Env      providers.BuildkiteEnv
			Detected bool
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
		GitRepository struct {
			Branch        string
			CommitSha     string
			CommitMessage string
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

		// Github Actions
		//
		if err := viper.BindEnv("ci.github.detected", "GITHUB_ACTIONS"); err != nil {
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

		// Buildkite
		//
		if err := viper.BindEnv("ci.buildkite.detected", "BUILDKITE"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteJobId", "BUILDKITE_JOB_ID"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteLabel", "BUILDKITE_LABEL"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteParallelJob", "BUILDKITE_PARALLEL_JOB"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteParallelJobCount", "BUILDKITE_PARALLEL_JOB_COUNT"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteRetryCount", "BUILDKITE_RETRY_COUNT"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteBuildCreatorEmail", "BUILDKITE_BUILD_CREATOR_EMAIL"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteBuildId", "BUILDKITE_BUILD_ID"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteBuildUrl", "BUILDKITE_BUILD_URL"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteBranch", "BUILDKITE_BRANCH"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteMessage", "BUILDKITE_MESSAGE"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteCommit", "BUILDKITE_COMMIT"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteRepo", "BUILDKITE_REPO"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		if err := viper.BindEnv("ci.buildkite.env.buildkiteOrganizationSlug", "BUILDKITE_ORGANIZATION_SLUG"); err != nil {
			err = errors.NewConfigurationError("unable to read from environment: %s", err)
			initializationErrors = append(initializationErrors, err)
		}

		bindGitRepoConfig()
	})
}

func bindGitRepoConfig() {
	path, err := os.Getwd()
	if err != nil {
		return
	}

	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return
	}

	head, err := repo.Head()
	if err != nil {
		return
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return
	}

	viper.Set("vcs.gitRepository.branch", head.Name().Short())
	viper.Set("vcs.gitRepository.commitSha", commit.Hash.String())
	viper.Set("vcs.gitRepository.commitMessage", commit.Message)
}
