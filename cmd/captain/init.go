package main

import (
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/logging"
	"github.com/rwx-research/captain-cli/internal/parsers"
)

func initCLIService(cmd *cobra.Command, args []string) error {
	var cfg config

	if err := viper.Unmarshal(&cfg); err != nil {
		// TODO: Check if this viper error are ok to present to end-users
		return errors.NewConfigurationError("unable to parse configuration: %s", err)
	}

	logger := logging.NewProductionLogger()
	if cfg.Debug {
		logger = logging.NewDebugLogger()
	}

	owner, repository := path.Split(cfg.VCS.Github.Repository)

	apiClient, err := api.NewClient(api.ClientConfig{
		AccountName:    strings.TrimSuffix(owner, "/"),
		Debug:          cfg.Debug,
		Host:           cfg.Captain.Host,
		Insecure:       cfg.Insecure,
		JobName:        cfg.CI.Github.Job.Name,
		JobMatrix:      cfg.CI.Github.Job.Matrix,
		Log:            logger,
		RunAttempt:     cfg.CI.Github.Run.Attempt,
		RunID:          cfg.CI.Github.Run.ID,
		RepositoryName: repository,
		Token:          cfg.Captain.Token,
	})
	if err != nil {
		return errors.WithMessage("unable to create API client: %w", err)
	}

	captain = cli.Service{
		API:        apiClient,
		Log:        logger,
		FileSystem: fs.Local{},
		TaskRunner: exec.Local{},
		Parsers:    []cli.Parser{new(parsers.JUnit), new(parsers.Jest), new(parsers.RSpecV3), new(parsers.XUnitDotNetV2)},
	}

	return nil
}

// unsafeInitParsingOnly initializes an incomplete `captain` CLI service. This service is sufficient for running
// `captain parse`, but not for any other operation.
// It is considered unsafe since the captain CLI service might still expect a configured API at one point.
func unsafeInitParsingOnly(cmd *cobra.Command, args []string) error {
	var cfg config

	if err := viper.Unmarshal(&cfg); err != nil {
		// TODO: Check if this viper error are ok to present to end-users
		return errors.NewConfigurationError("unable to parse configuration: %s", err)
	}

	logger := logging.NewProductionLogger()
	if cfg.Debug {
		logger = logging.NewDebugLogger()
	}

	captain = cli.Service{
		Log:        logger,
		FileSystem: fs.Local{},
		Parsers:    []cli.Parser{new(parsers.JUnit), new(parsers.Jest), new(parsers.RSpecV3), new(parsers.XUnitDotNetV2)},
	}

	return nil
}
