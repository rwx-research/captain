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

var (
	captain       cli.Service
	debug         bool
	githubJobName string
	insecure      bool
	suiteName     string
	version       bool

	rootCmd = &cobra.Command{
		Use:               "captain",
		Short:             "Captain makes your builds faster and more reliable",
		Long:              descriptionCaptain,
		PersistentPreRunE: initCLIService,
		SilenceErrors:     true, // Errors are manually printed in 'main'
		SilenceUsage:      true, // Disables usage text on error
		RunE: func(cmd *cobra.Command, args []string) error {
			if version {
				captain.PrintVersion()
				return nil
			}

			return errors.Wrap(cmd.Usage())
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&suiteName, "suite-name", "", "the name of the build- or test-suite")

	rootCmd.PersistentFlags().StringVar(&githubJobName, "github-job-name", "", "the name of the current Github Job")
	if err := viper.BindPFlag("ci.github.job", rootCmd.PersistentFlags().Lookup("github-job-name")); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	if err := rootCmd.PersistentFlags().MarkHidden("debug"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}
	if err := viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug")); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "disable TLS for the API")
	if err := rootCmd.PersistentFlags().MarkHidden("insecure"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}
	if err := viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure")); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.Flags().BoolVar(&version, "version", false, "print the Captain CLI version")
	if err := rootCmd.Flags().MarkHidden("version"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

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
		JobName:        cfg.CI.Github.Job,
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
