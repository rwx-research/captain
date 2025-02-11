package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v7"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/backend/remote"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
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

// adds config to cmd's context
func setConfigContext(cmd *cobra.Command, cfg Config) error {
	if _, err := getConfig(cmd); err == nil {
		return errors.NewInternalError("Tried to set config on the command but it was already set. This should never happen!")
	}

	ctx := context.WithValue(cmd.Context(), configKey, cfg)
	cmd.SetContext(ctx)
	return nil
}

const (
	captainDirectory    = ".captain"
	configFileName      = "config"
	flakesFileName      = "flakes.yaml"
	quarantinesFileName = "quarantines.yaml"
	timingsFileName     = "timings.yaml"
)

var configFileExtensions = []string{"yaml", "yml"}

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
		possibleConfigFilePaths := make([]string, 0, 2)
		errs := make([]error, 0, 2)

		for _, extension := range configFileExtensions {
			configFilePath, err := findInParentDir(
				filepath.Join(captainDirectory, fmt.Sprintf("%s.%s", configFileName, extension)),
			)

			if err == nil {
				possibleConfigFilePaths = append(possibleConfigFilePaths, configFilePath)
			} else {
				errs = append(errs, err)
			}
		}

		if len(possibleConfigFilePaths) > 1 {
			return cfg, errors.NewConfigurationError(
				"Unable to identify configuration file",
				fmt.Sprintf(
					"Captain found multiple configuration files in your environment: %s\n",
					strings.Join(possibleConfigFilePaths, ", "),
				),
				"Please make sure only one config file is present in your environment or explicitly specify "+
					"one using the '--config-file' flag.",
			)
		}

		if len(possibleConfigFilePaths) == 0 {
			for _, err := range errs {
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
		} else {
			cliArgs.RootCliArgs.configFilePath = possibleConfigFilePaths[0]
		}
	}

	if cliArgs.RootCliArgs.configFilePath != "" {
		fd, err := os.Open(cliArgs.RootCliArgs.configFilePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return cfg, errors.Wrap(err, fmt.Sprintf("unable to open config file %q", cliArgs.RootCliArgs.configFilePath))
			}
		} else {
			decoder := yaml.NewDecoder(fd)
			decoder.KnownFields(true)
			if err = decoder.Decode(&cfg.ConfigFile); err != nil {
				typeError := new(yaml.TypeError)
				if errors.As(err, &typeError) {
					err = errors.NewConfigurationError(
						"Parsing Error",
						strings.Join(typeError.Errors, "\n"),
						"Please refer to the documentation at https://www.rwx.com/docs/captain/cli-configuration/config-yaml for the"+
							" correct config file syntax.",
					)
				}

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

	cfg = bindRootCmdFlags(cfg, cliArgs.RootCliArgs)
	cfg = bindFrameworkFlags(cfg, cliArgs.frameworkParams, cliArgs.RootCliArgs.suiteID)
	cfg = bindRunCmdFlags(cfg, cliArgs)

	if err = setConfigContext(cmd, cfg); err != nil {
		return cfg, errors.WithStack(err)
	}

	return cfg, nil
}

func getRecipes(logger *zap.SugaredLogger, cfg Config) (map[string]v1.TestIdentityRecipe, error) {
	var recipeBuffer []byte

	existingCaptainDir, err := findInParentDir(captainDirectory)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		if err = os.Mkdir(captainDirectory, 0o755); err != nil {
			return nil, errors.WithStack(err)
		}

		existingCaptainDir = captainDirectory
	}

	recipesFile, err := findInParentDir(filepath.Join(captainDirectory, "recipes.json"))
	if err == nil {
		recipeBuffer, err = os.ReadFile(recipesFile)
	}
	if err != nil {
		if recipesFile == "" {
			recipesFile = filepath.Join(existingCaptainDir, "recipes.json")
		}

		client, err := remote.NewClient(remote.ClientConfig{
			Debug:    cfg.Output.Debug,
			Host:     cfg.Cloud.APIHost,
			Insecure: cfg.Cloud.Insecure,
			Log:      logger,
			Token:    "none", // Can't be empty. We rely on implementation details here that `GetIdentityRecipes` will not use it
		})
		if err != nil {
			return nil, errors.Wrap(err, "Unable to initialize API client")
		}

		recipeBuffer, err = client.GetIdentityRecipes(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "Unable to fetch test identity recipes from API")
		}

		if err = os.WriteFile(recipesFile, recipeBuffer, 0o600); err != nil {
			logger.Warnf("unable to cache identity recipes on disk: %s", err.Error())
		}
	}

	type IdentityRecipe struct {
		Language string
		Kind     string
		Recipe   struct {
			Components []string
			Strict     bool
		}
	}

	recipeList := []IdentityRecipe{}
	if err := json.Unmarshal(recipeBuffer, &recipeList); err != nil {
		return nil, errors.NewInternalError("unable to parse identiy recipes: %s", err.Error())
	}

	recipes := make(map[string]v1.TestIdentityRecipe)
	for _, identityRecipe := range recipeList {
		recipes[v1.CoerceFramework(identityRecipe.Language, identityRecipe.Kind).String()] = v1.TestIdentityRecipe{
			Components: identityRecipe.Recipe.Components,
			Strict:     identityRecipe.Recipe.Strict,
		}
	}

	return recipes, nil
}
