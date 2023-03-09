package main

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/logging"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/providers"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

var mutuallyExclusiveParsers []parsing.Parser = []parsing.Parser{
	new(parsing.DotNetxUnitParser),
	new(parsing.GoGinkgoParser),
	new(parsing.GoTestParser),
	new(parsing.JavaScriptCypressParser),
	new(parsing.JavaScriptJestParser),
	new(parsing.JavaScriptKarmaParser),
	new(parsing.JavaScriptMochaParser),
	new(parsing.JavaScriptPlaywrightParser),
	new(parsing.PythonPytestParser),
	new(parsing.RubyCucumberParser),
	new(parsing.RubyRSpecParser),
}

var frameworkParsers map[v1.Framework][]parsing.Parser = map[v1.Framework][]parsing.Parser{
	v1.DotNetxUnitFramework:          {new(parsing.DotNetxUnitParser)},
	v1.ElixirExUnitFramework:         {new(parsing.ElixirExUnitParser)},
	v1.GoGinkgoFramework:             {new(parsing.GoGinkgoParser)},
	v1.GoTestFramework:               {new(parsing.GoTestParser)},
	v1.JavaScriptCypressFramework:    {new(parsing.JavaScriptCypressParser)},
	v1.JavaScriptJestFramework:       {new(parsing.JavaScriptJestParser)},
	v1.JavaScriptKarmaFramework:      {new(parsing.JavaScriptKarmaParser)},
	v1.JavaScriptMochaFramework:      {new(parsing.JavaScriptMochaParser)},
	v1.JavaScriptPlaywrightFramework: {new(parsing.JavaScriptPlaywrightParser)},
	v1.PHPUnitFramework:              {new(parsing.PHPUnitParser)},
	v1.PythonPytestFramework:         {new(parsing.PythonPytestParser)},
	v1.PythonUnitTestFramework:       {new(parsing.PythonUnitTestParser)},
	v1.RubyCucumberFramework:         {new(parsing.RubyCucumberParser)},
	v1.RubyMinitestFramework:         {new(parsing.RubyMinitestParser)},
	v1.RubyRSpecFramework:            {new(parsing.RubyRSpecParser)},
}

var genericParsers []parsing.Parser = []parsing.Parser{
	new(parsing.RWXParser),
	new(parsing.JUnitParser),
}

// TODO: It looks like errors returned from this function are not getting logged correctly
func initCLIService(cmd *cobra.Command, args []string) error {
	var err error

	cfg, err = InitConfig(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	logger := logging.NewProductionLogger()
	if cfg.Output.Debug {
		logger = logging.NewDebugLogger()
	}

	providerAdapter, err := MakeProviderAdapter(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to construct provider adapter")
	}

	apiClient, err := api.NewClient(api.ClientConfig{
		Debug:    cfg.Output.Debug,
		Host:     cfg.Cloud.APIHost,
		Insecure: cfg.Cloud.Insecure,
		Log:      logger,
		Token:    cfg.Secrets.APIToken,
		Provider: providerAdapter,
	})
	if err != nil {
		return errors.Wrap(err, "unable to create API client")
	}

	var frameworkKind, frameworkLanguage string
	if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
		frameworkKind = suiteConfig.Results.Framework
		frameworkLanguage = suiteConfig.Results.Language
	}

	parseConfig := parsing.Config{
		ProvidedFrameworkKind:     frameworkKind,
		ProvidedFrameworkLanguage: frameworkLanguage,
		MutuallyExclusiveParsers:  mutuallyExclusiveParsers,
		FrameworkParsers:          frameworkParsers,
		GenericParsers:            genericParsers,
		Logger:                    logger,
	}
	if err := parseConfig.Validate(); err != nil {
		return errors.Wrap(err, "invalid parser config")
	}

	captain = cli.Service{
		API:         apiClient,
		Log:         logger,
		FileSystem:  fs.Local{},
		TaskRunner:  exec.Local{},
		ParseConfig: parseConfig,
	}

	return nil
}

func MakeProviderAdapter(cfg Config) (providers.Provider, error) {
	switch {
	case cfg.GitHub.Detected:
		return MakeGithubProvider(cfg)
	case cfg.Buildkite.Detected:
		return MakeBuildkiteProvider(cfg)
	case cfg.CircleCI.Detected:
		return MakeCircleciProvider(cfg)
	case cfg.GitLab.Detected:
		return MakeGitLabProvider(cfg)
	default:
		return providers.NullProvider{}, errors.NewConfigurationError("Failed to detect supported CI context")
	}
}

func MakeBuildkiteProvider(cfg Config) (providers.BuildkiteProvider, error) {
	return providers.BuildkiteProvider{
		Branch:            cfg.Buildkite.Branch,
		BuildCreatorEmail: cfg.Buildkite.BuildCreatorEmail,
		BuildID:           cfg.Buildkite.BuildID,
		BuildURL:          cfg.Buildkite.BuildURL,
		Commit:            cfg.Buildkite.Commit,
		JobID:             cfg.Buildkite.JobID,
		Label:             cfg.Buildkite.Label,
		Message:           cfg.Buildkite.Message,
		OrganizationSlug:  cfg.Buildkite.OrganizationSlug,
		ParallelJob:       cfg.Buildkite.ParallelJob,
		ParallelJobCount:  cfg.Buildkite.ParallelJobCount,
		Repo:              cfg.Buildkite.Repo,
		RetryCount:        cfg.Buildkite.RetryCount,
	}, nil
}

func MakeCircleciProvider(cfg Config) (providers.CircleciProvider, error) {
	return providers.CircleciProvider{
		BuildNum:         cfg.CircleCI.BuildNum,
		BuildURL:         cfg.CircleCI.BuildURL,
		JobName:          cfg.CircleCI.Job,
		ParallelJobIndex: cfg.CircleCI.NodeIndex,
		ParallelJobTotal: cfg.CircleCI.NodeTotal,
		RepoAccountName:  cfg.CircleCI.ProjectUsername,
		RepoName:         cfg.CircleCI.ProjectReponame,
		RepoURL:          cfg.CircleCI.RepositoryURL,
		BranchName:       cfg.CircleCI.Branch,
		CommitSha:        cfg.CircleCI.Sha1,
		AttemptedBy:      cfg.CircleCI.Username,
	}, nil
}

func MakeGitLabProvider(cfg Config) (providers.GitLabCIProvider, error) {
	return providers.GitLabCIProvider{
		JobName:       cfg.GitLab.CIJobName,
		JobStage:      cfg.GitLab.CIJobStage,
		JobID:         cfg.GitLab.CIJobID,
		PipelineID:    cfg.GitLab.CIPipelineID,
		JobURL:        cfg.GitLab.CIJobURL,
		PipelineURL:   cfg.GitLab.CIPipelineURL,
		AttemptedBy:   cfg.GitLab.GitlabUserLogin,
		NodeIndex:     cfg.GitLab.CINodeIndex,
		NodeTotal:     cfg.GitLab.CINodeTotal,
		RepoPath:      cfg.GitLab.CIProjectPath,
		ProjectURL:    cfg.GitLab.CIProjectURL,
		CommitSHA:     cfg.GitLab.CICommitSHA,
		CommitAuthor:  cfg.GitLab.CICommitAuthor,
		BranchName:    cfg.GitLab.CICommitBranch,
		CommitMessage: cfg.GitLab.CICommitMessage,
		APIURL:        cfg.GitLab.CIAPIV4URL,
	}, nil
}

func MakeGithubProvider(cfg Config) (providers.GithubProvider, error) {
	branchName := cfg.GitHub.RefName
	if cfg.GitHub.EventName == "pull_request" {
		branchName = cfg.GitHub.HeadRef
	}

	eventPayloadData := struct {
		HeadCommit struct {
			Message string `json:"message"`
		} `json:"head_commit"`
	}{}

	file, err := os.Open(cfg.GitHub.EventPath)
	if err != nil && !os.IsNotExist(err) {
		return providers.GithubProvider{}, errors.Wrap(err, "unable to open event payload file")
	} else if err == nil {
		if err := json.NewDecoder(file).Decode(&eventPayloadData); err != nil {
			return providers.GithubProvider{}, errors.Wrap(err, "failed to decode event payload data")
		}
	}

	attemptedBy := cfg.GitHub.TriggeringActor
	if attemptedBy == "" {
		attemptedBy = cfg.GitHub.ExecutingActor
	}

	owner, repository := path.Split(cfg.GitHub.Repository)

	return providers.GithubProvider{
		AccountName:    strings.TrimSuffix(owner, "/"),
		AttemptedBy:    attemptedBy,
		BranchName:     branchName,
		CommitMessage:  eventPayloadData.HeadCommit.Message,
		CommitSha:      cfg.GitHub.CommitSha,
		JobName:        cfg.GitHub.Name,
		JobMatrix:      cfg.GitHub.Matrix,
		RunAttempt:     cfg.GitHub.Attempt,
		RunID:          cfg.GitHub.ID,
		RepositoryName: repository,
	}, nil
}

// unsafeInitParsingOnly initializes an incomplete `captain` CLI service. This service is sufficient for running
// `captain parse`, but not for any other operation.
// It is considered unsafe since the captain CLI service might still expect a configured API at one point.
func unsafeInitParsingOnly(cmd *cobra.Command, args []string) error {
	var err error

	cfg, err = InitConfig(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	logger := logging.NewProductionLogger()
	if cfg.Output.Debug {
		logger = logging.NewDebugLogger()
	}

	var frameworkKind, frameworkLanguage string
	if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
		frameworkKind = suiteConfig.Results.Framework
		frameworkLanguage = suiteConfig.Results.Language
	}

	parseConfig := parsing.Config{
		ProvidedFrameworkKind:     frameworkKind,
		ProvidedFrameworkLanguage: frameworkLanguage,
		MutuallyExclusiveParsers:  mutuallyExclusiveParsers,
		FrameworkParsers:          frameworkParsers,
		GenericParsers:            genericParsers,
		Logger:                    logger,
	}
	if err := parseConfig.Validate(); err != nil {
		return errors.Wrap(err, "invalid parser config")
	}

	captain = cli.Service{
		Log:         logger,
		FileSystem:  fs.Local{},
		ParseConfig: parseConfig,
	}

	return nil
}
