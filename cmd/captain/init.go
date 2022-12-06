package main

import (
	"encoding/json"
	"os"
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
	"github.com/rwx-research/captain-cli/internal/parsing"
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

	branchName := cfg.VCS.Github.RefName
	if cfg.CI.Github.Run.EventName == "pull_request" {
		branchName = cfg.VCS.Github.HeadRef
	}

	eventPayloadData := struct {
		HeadCommit struct {
			Message string `json:"message"`
		} `json:"head_commit"`
	}{}

	file, err := os.Open(cfg.CI.Github.Run.EventPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "unable to open event payload file")
	} else if err == nil {
		if err := json.NewDecoder(file).Decode(&eventPayloadData); err != nil {
			return errors.Wrap(err, "failed to decode event payload data")
		}
	}

	attemptedBy := cfg.CI.Github.Run.TriggeringActor
	if attemptedBy == "" {
		attemptedBy = cfg.CI.Github.Run.ExecutingActor
	}

	owner, repository := path.Split(cfg.VCS.Github.Repository)

	apiClient, err := api.NewClient(api.ClientConfig{
		AccountName:    strings.TrimSuffix(owner, "/"),
		AttemptedBy:    attemptedBy,
		BranchName:     branchName,
		CommitMessage:  eventPayloadData.HeadCommit.Message,
		CommitSha:      cfg.VCS.Github.CommitSha,
		Debug:          cfg.Debug,
		Host:           cfg.Captain.Host,
		Insecure:       cfg.Insecure,
		JobName:        cfg.CI.Github.Job.Name,
		JobMatrix:      cfg.CI.Github.Job.Matrix,
		Log:            logger,
		Provider:       "github",
		RunAttempt:     cfg.CI.Github.Run.Attempt,
		RunID:          cfg.CI.Github.Run.ID,
		RepositoryName: repository,
		Token:          cfg.Captain.Token,
	})
	if err != nil {
		return errors.Wrap(err, "unable to create API client")
	}

	captain = cli.Service{
		API:        apiClient,
		Log:        logger,
		FileSystem: fs.Local{},
		TaskRunner: exec.Local{},
		Parsers: []parsing.Parser{
			new(parsing.RWXParser),
			new(parsing.DotNetxUnitParser),
			new(parsing.JavaScriptCypressParser),
			new(parsing.JavaScriptJestParser),
			new(parsing.RubyRSpecParser),
			new(parsing.JUnitParser),
		},
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
		Parsers: []parsing.Parser{
			new(parsing.RWXParser),
			new(parsing.RubyCucumberParser),
			new(parsing.DotNetxUnitParser),
			new(parsing.JavaScriptCypressParser),
			new(parsing.JavaScriptJestParser),
			new(parsing.RubyRSpecParser),
			new(parsing.JUnitParser),
		},
	}

	return nil
}
