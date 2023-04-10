package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v7"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

// Config is the internal representation of the configuration.
type Config struct {
	cli.ConfigFile

	ProvidersEnv providers.Env

	Secrets struct {
		APIToken string `env:"RWX_ACCESS_TOKEN"`
	}
}

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
			cfg.TestSuites = make(map[string]cli.SuiteConfig)
		}

		cfg.TestSuites[cliArgs.RootCliArgs.suiteID] = cli.SuiteConfig{}
	}

	cfg = detectRootDir(cfg, cliArgs.RootCliArgs)
	cfg = applyRootDirToConfigFile(cfg)

	// These must come after the root dir and applying the root dir to the config file.
	// The root dir only affects relative paths in the config file.
	cfg = bindRootCmdFlags(cfg, cliArgs.RootCliArgs)
	cfg = bindFrameworkFlags(cfg, cliArgs.frameworkParams, cliArgs.RootCliArgs.suiteID)
	cfg = bindRunCmdFlags(cfg, cliArgs)

	return cfg, nil
}

// Detects the root-dir relative to the config file path.
// If the config file is found within a .captain subdirectory, the root-dir
// is detected to be the directory containing the .captain directory.
// Otherwise, the root-dir is "" which, in effect, acts as the working dir.
func detectRootDir(cfg Config, rootCliArgs rootCliArgs) Config {
	if rootCliArgs.rootDir != "" {
		cfg.RootDir = rootCliArgs.rootDir
		return cfg
	}

	if cfg.RootDir != "" {
		return cfg
	}

	if !strings.HasSuffix(rootCliArgs.configFilePath, filepath.Join(captainDirectory, configFileName)) {
		// Leave it as "" to use the working dir
		return cfg
	}

	detectedRootDir := strings.TrimSuffix(rootCliArgs.configFilePath, filepath.Join(captainDirectory, configFileName))
	cfg.RootDir = filepath.Clean(detectedRootDir)
	return cfg
}

func applyRootDirToConfigFile(cfg Config) Config {
	for suiteID, suiteConfig := range cfg.TestSuites {
		for reporter, path := range suiteConfig.Output.Reporters {
			if !filepath.IsAbs(path) {
				suiteConfig.Output.Reporters[reporter] = filepath.Join(cfg.RootDir, path)
			}
		}

		if suiteConfig.Results.Path != "" && !filepath.IsAbs(suiteConfig.Results.Path) {
			suiteConfig.Results.Path = filepath.Join(cfg.RootDir, suiteConfig.Results.Path)
		}

		if suiteConfig.Retries.IntermediateArtifactsPath != "" &&
			!filepath.IsAbs(suiteConfig.Retries.IntermediateArtifactsPath) {
			suiteConfig.Retries.IntermediateArtifactsPath = filepath.Join(
				cfg.RootDir,
				suiteConfig.Retries.IntermediateArtifactsPath,
			)
		}

		cfg.TestSuites[suiteID] = suiteConfig
	}

	return cfg
}
