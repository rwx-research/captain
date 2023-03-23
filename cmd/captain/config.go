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
	"github.com/rwx-research/captain-cli/internal/providers"
)

const (
	configFileName      = ".captain/config.yaml"
	flakyTestsFileName  = ".captain/flaky-tests.json"
	testTimingsFileName = ".captain/test-timings.json"
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
func InitConfig(cmd *cobra.Command) (cfg Config, err error) {
	if configFilePath == "" {
		configFilePath, err = findInParentDir(configFileName)
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
