package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/caarlos0/env/v7"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// Config is the internal representation of the configuration.
type Config struct {
	ConfigFile

	GitHub struct {
		Detected        bool   `env:"GITHUB_ACTIONS"`
		Name            string `env:"GITHUB_JOB"`
		Matrix          string
		Attempt         string `env:"GITHUB_RUN_ATTEMPT"`
		ID              string `env:"GITHUB_RUN_ID"`
		EventName       string `env:"GITHUB_EVENT_NAME"`
		EventPath       string `env:"GITHUB_EVENT_PATH"`
		ExecutingActor  string `env:"GITHUB_ACTOR"`
		TriggeringActor string `env:"GITHUB_TRIGGERING_ACTOR"`
		Repository      string `env:"GITHUB_REPOSITORY"`
		RefName         string `env:"GITHUB_REF_NAME"`
		HeadRef         string `env:"GITHUB_HEAD_REF"`
		CommitSha       string `env:"GITHUB_SHA"`
	}
	Buildkite struct {
		Detected          bool   `env:"BUILDKITE"`
		Branch            string `env:"BUILDKITE_BRANCH"`
		BuildCreatorEmail string `env:"BUILDKITE_BUILD_CREATOR_EMAIL"`
		BuildID           string `env:"BUILDKITE_BUILD_ID"`
		BuildURL          string `env:"BUILDKITE_BUILD_URL"`
		Commit            string `env:"BUILDKITE_COMMIT"`
		JobID             string `env:"BUILDKITE_JOB_ID"`
		Label             string `env:"BUILDKITE_LABEL"`
		Message           string `env:"BUILDKITE_MESSAGE"`
		OrganizationSlug  string `env:"BUILDKITE_ORGANIZATION_SLUG"`
		ParallelJob       string `env:"BUILDKITE_PARALLEL_JOB"`
		ParallelJobCount  string `env:"BUILDKITE_PARALLEL_JOB_COUNT"`
		Repo              string `env:"BUILDKITE_REPO"`
		RetryCount        string `env:"BUILDKITE_RETRY_COUNT"`
	}
	CircleCI struct {
		Detected        bool   `env:"CIRCLECI"`
		BuildNum        string `env:"CIRCLE_BUILD_NUM"`
		BuildURL        string `env:"CIRCLE_BUILD_URL"`
		Job             string `env:"CIRCLE_JOB"`
		NodeIndex       string `env:"CIRCLE_NODE_INDEX"`
		NodeTotal       string `env:"CIRCLE_NODE_TOTAL"`
		ProjectReponame string `env:"CIRCLE_PROJECT_REPONAME"`
		ProjectUsername string `env:"CIRCLE_PROJECT_USERNAME"`
		RepositoryURL   string `env:"CIRCLE_REPOSITORY_URL"`
		Branch          string `env:"CIRCLE_BRANCH"`
		Sha1            string `env:"CIRCLE_SHA1"`
		Username        string `env:"CIRCLE_USERNAME"`
	}
	GitLab struct {
		// see https://docs.gitlab.com/ee/ci/variables/predefined_variables.html
		// gitlab/runner version all/all
		Detected bool `env:"GITLAB_CI"`

		// Build Info
		// gitlab/runner version 9.0/0.5
		CIJobName string `env:"CI_JOB_NAME"`
		// gitlab/runner version 9.0/0.5
		CIJobStage string `env:"CI_JOB_STAGE"`
		// gitlab/runner version 9.0/all
		CIJobID string `env:"CI_JOB_ID"`
		// gitlab/runner version 8.10/all
		CIPipelineID string `env:"CI_PIPELINE_ID"`
		// gitlab/runner version 11.1/0.5
		CIJobURL string `env:"CI_JOB_URL"`
		// gitlab/runner version 11.1/0.5
		CIPipelineURL string `env:"CI_PIPELINE_URL"`
		// gitlab/runner version 10.0/all
		GitlabUserLogin string `env:"GITLAB_USER_LOGIN"`
		// gitlab/runner version 11.5/all
		CINodeTotal string `env:"CI_NODE_TOTAL"`
		// gitlab/runner version 11.5/all
		CINodeIndex string `env:"CI_NODE_INDEX"`

		// Repo Info
		// gitlab/runner version 8.10/0.5
		CIProjectPath string `env:"CI_PROJECT_PATH"`
		// gitlab/runner version 8.10/0.5
		CIProjectURL string `env:"CI_PROJECT_URL"`

		// Commit Info
		// gitlab/runner version 9.0/all
		CICommitSHA string `env:"CI_COMMIT_SHA"`
		// gitlab/runner version 13.11/all
		CICommitAuthor string `env:"CI_COMMIT_AUTHOR"`
		// gitlab/runner version 12.6/0.5
		CICommitBranch string `env:"CI_COMMIT_BRANCH"`
		// gitlab/runner version 10.8/all
		CICommitMessage string `env:"CI_COMMIT_MESSAGE"`

		// Consider in the future checking CI_SERVER_VERSION >= 13.2 (the newest version with all of these fields)
		// gitlab/runner version 11.7/all
		CIAPIV4URL string `env:"CI_API_V4_URL"`
	}
	Secrets struct {
		APIToken string `env:"RWX_ACCESS_TOKEN"`
	}
}

// configFile holds all options that can be set over the config file
type ConfigFile struct {
	Cloud struct {
		APIHost  string `yaml:"api-host" env:"CAPTAIN_HOST"`
		Insecure bool
	}
	Flags  map[string]any
	Output struct {
		Debug bool
		Quiet bool
	}
	TestSuites map[string]SuiteConfig `yaml:"test-suites"`
}

// SuiteConfig holds options that can be customized per suite
type SuiteConfig struct {
	Command string
	Output  struct {
		PrintSummary bool `yaml:"print-summary"`
		Reporters    map[string]string
	}
	Results struct {
		Framework string
		Language  string
		Path      string
	}
	Retries struct {
		Attempts          int
		Command           string
		FailFast          bool `yaml:"fail-fast"`
		FlakyAttempts     int  `yaml:"flaky-attempts"`
		MaxTests          string
		PostRetryCommands []string `yaml:"post-retry-commands"`
		PreRetryCommands  []string `yaml:"pre-retry-commands"`
	}
}

// findConfigFilePath starts at the current working directory and walk up to the root, trying
// to find `.captain/config.yaml`
func findConfigFilePath() (string, error) {
	var match string
	var walk func(string, string) error

	walk = func(base, root string) error {
		if base == root {
			return errors.WithStack(os.ErrNotExist)
		}

		match = path.Join(base, ".captain/config.yaml")

		info, err := os.Stat(match)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.WithStack(err)
		}

		if info != nil {
			return nil
		}

		return walk(filepath.Dir(base), root)
	}

	pwd, err := os.Getwd()
	if err != nil {
		return "", errors.WithStack(err)
	}

	volumeName := filepath.VolumeName(pwd)
	if volumeName == "" {
		volumeName = string(os.PathSeparator)
	}

	if err := walk(pwd, volumeName); err != nil {
		return "", errors.WithStack(err)
	}

	return match, nil
}

// InitConfig reads our configuration from the system.
// Environment variables take precedence over a config file.
// Flags take precedence over all other options.
func InitConfig(cmd *cobra.Command) (cfg Config, err error) {
	if configFilePath == "" {
		configFilePath, err = findConfigFilePath()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return cfg, errors.NewConfigurationError("unable to determine config file path: %s", err.Error())
		}
	}

	if configFilePath != "" {
		fd, err := os.Open(configFilePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return cfg, errors.Wrap(err, fmt.Sprintf("unable to open config file %q", configFilePath))
			}
		} else {
			if err = yaml.NewDecoder(fd).Decode(&cfg.ConfigFile); err != nil {
				return cfg, errors.Wrap(err, "unable to parse config file")
			}
		}
	}

	for name, value := range cfg.Flags {
		if err := cmd.Flags().Set(name, fmt.Sprintf("%v", value)); err != nil {
			return cfg, errors.Wrap(err, fmt.Sprintf("unable to set flag %q", name))
		}
	}

	if err = env.Parse(&cfg); err != nil {
		return cfg, errors.Wrap(err, "unable to parse environment variables")
	}

	if _, ok := cfg.TestSuites[suiteID]; !ok {
		if cfg.TestSuites == nil {
			cfg.TestSuites = make(map[string]SuiteConfig)
		}

		cfg.TestSuites[suiteID] = SuiteConfig{}
	}

	cfg = bindRootCmdFlags(cfg)
	cfg = bindFrameworkFlags(cfg)
	cfg = bindRunCmdFlags(cfg)

	return cfg, nil
}
