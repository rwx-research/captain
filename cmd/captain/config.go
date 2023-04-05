package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/caarlos0/env/v7"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

type contextKey string

var configKey = contextKey("captainConfig")

func getConfig(cmd *cobra.Command) (Config, error) {
	val := cmd.Context().Value(configKey)
	if val == nil {
		return Config{}, errors.NewInternalError(
			"Tried to fetch config from the command but it wasn't set. This should never happen!")
	}

	cfg, ok := val.(Config)
	if !ok {
		return Config{}, errors.NewInternalError(
			"Tried to fetch config from the command but it was of the wrong type. This should never happen!")
	}

	return cfg, nil
}

func setConfig(cmd *cobra.Command, cfg Config) error {
	if _, err := getConfig(cmd); err == nil {
		return errors.NewInternalError("Tried to set config on the command but it was already set. This should never happen!")
	}

	ctx := context.WithValue(cmd.Context(), configKey, cfg)
	cmd.SetContext(ctx)
	return nil
}

const (
	captainDirectory    = ".captain"
	configFileName      = "config.yaml"
	flakesFileName      = "flakes.yaml"
	quarantinesFileName = "quarantines.yaml"
	timingsFileName     = "timings.yaml"
)

// Config is the internal representation of the configuration.
type Config struct {
	ConfigFile

	ProvidersEnv providers.Env

	Secrets struct {
		APIToken string `env:"RWX_ACCESS_TOKEN"`
	}
}

// configFile holds all options that can be set over the config file
type ConfigFile struct {
	Cloud struct {
		APIHost  string `yaml:"api-host" env:"CAPTAIN_HOST"`
		Disabled bool
		Insecure bool
	}
	Flags  map[string]any
	Output struct {
		Debug bool
	}
	TestSuites map[string]SuiteConfig `yaml:"test-suites"`
}

// SuiteConfig holds options that can be customized per suite
type SuiteConfig struct {
	Command           string
	FailOnUploadError bool `yaml:"fail-on-upload-error"`
	Output            struct {
		PrintSummary bool `yaml:"print-summary"`
		Reporters    map[string]string
		Quiet        bool
	}
	Results struct {
		Framework string
		Language  string
		Path      string
	}
	Retries struct {
		Attempts                  int
		Command                   string
		FailFast                  bool `yaml:"fail-fast"`
		FlakyAttempts             int  `yaml:"flaky-attempts"`
		MaxTests                  string
		PostRetryCommands         []string `yaml:"post-retry-commands"`
		PreRetryCommands          []string `yaml:"pre-retry-commands"`
		IntermediateArtifactsPath string   `yaml:"intermediate-artifacts-path"`
	}
}

// findInParentDir starts at the current working directory and walk up to the root, trying
// to find the specified fileName
func findInParentDir(fileName string) (string, error) {
	var match string
	var walk func(string, string) error

	walk = func(base, root string) error {
		if base == root {
			return errors.WithStack(os.ErrNotExist)
		}

		match = path.Join(base, fileName)

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
func InitConfig(cmd *cobra.Command, cliArgs CliArgs) (cfg Config, err error) {
	if cliArgs.RootCliArgs.configFilePath == "" {
		cliArgs.RootCliArgs.configFilePath, err = findInParentDir(filepath.Join(captainDirectory, configFileName))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return cfg, errors.NewConfigurationError(
				"Unable to read configuration file",
				fmt.Sprintf(
					"The following system error occurred while attempting to read the config file at %q: %s",
					cliArgs.RootCliArgs.configFilePath, err.Error(),
				),
				"Please make sure that Captain has the correct permissions to access the config file.",
			)
		}
	}

	if cliArgs.RootCliArgs.configFilePath != "" {
		fd, err := os.Open(cliArgs.RootCliArgs.configFilePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return cfg, errors.Wrap(err, fmt.Sprintf("unable to open config file %q", cliArgs.RootCliArgs.configFilePath))
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

	if _, ok := cfg.TestSuites[cliArgs.RootCliArgs.suiteID]; !ok {
		if cfg.TestSuites == nil {
			cfg.TestSuites = make(map[string]SuiteConfig)
		}

		cfg.TestSuites[cliArgs.RootCliArgs.suiteID] = SuiteConfig{}
	}

	cfg = bindRootCmdFlags(cfg, cliArgs.RootCliArgs)
	cfg = bindFrameworkFlags(cfg, cliArgs.frameworkParams, cliArgs.RootCliArgs.suiteID)
	cfg = bindRunCmdFlags(cfg, cliArgs)

	return cfg, nil
}
