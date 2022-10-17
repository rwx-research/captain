package main

import (
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
)

var (
	captain cli.Service

	rootCmd = &cobra.Command{
		Use:               "captain",
		Short:             "Captain makes your builds faster and more reliable",
		Long:              descriptionCaptain,
		PersistentPreRunE: initCLIService,
		SilenceErrors:     true, // Errors are manually printed in 'main'
		SilenceUsage:      true, // Disables usage text on error
	}
)

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func initCLIService(cmd *cobra.Command, args []string) error {
	var cfg config

	if err := viper.Unmarshal(&cfg); err != nil {
		// TODO: Check if this viper error are ok to present to end-users
		return errors.ConfigurationError("unable to parse configuration: %s", err)
	}

	owner, repository := path.Split(cfg.VCS.Github.Repository)

	// TODO: Fine-tune logger. The defaults are very specific to SaaS applications, not CLIs
	//       There should also be a difference between production (default) & debug logging.
	logger, err := zap.NewDevelopment()
	if err != nil {
		return errors.InternalError("unable to create logger: %s", err)
	}

	apiClient, err := api.NewClient(api.ClientConfig{
		AccountName:    strings.TrimSuffix(owner, "/"),
		Host:           cfg.Captain.Host,
		Insecure:       cfg.Insecure,
		JobName:        cfg.CI.Github.Job,
		Log:            logger.Sugar(),
		RunAttempt:     cfg.CI.Github.Run.Attempt,
		RunID:          cfg.CI.Github.Run.ID,
		RepositoryName: repository,
		Token:          cfg.Captain.Token,
	})
	if err != nil {
		return xerrors.Errorf("unable to create API client: %w", err)
	}

	captain = cli.Service{
		API: apiClient,
		Log: logger.Sugar(),
	}

	return nil
}
