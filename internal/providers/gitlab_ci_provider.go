package providers

import (
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
)

type GitLabEnv struct {
	// see https://docs.gitlab.com/ee/ci/variables/predefined_variables.html
	// gitlab/runner version all/all
	Detected bool `env:"GITLAB_CI"`

	// Build Info
	JobName     string `env:"CI_JOB_NAME"`       // gitlab/runner version 9.0/0.5
	JobStage    string `env:"CI_JOB_STAGE"`      // gitlab/runner version 9.0/0.5
	JobID       string `env:"CI_JOB_ID"`         // gitlab/runner version 9.0/all
	PipelineID  string `env:"CI_PIPELINE_ID"`    // gitlab/runner version 8.10/all
	JobURL      string `env:"CI_JOB_URL"`        // gitlab/runner version 11.1/0.5
	PipelineURL string `env:"CI_PIPELINE_URL"`   // gitlab/runner version 11.1/0.5
	UserLogin   string `env:"GITLAB_USER_LOGIN"` // gitlab/runner version 10.0/all
	NodeTotal   string `env:"CI_NODE_TOTAL"`     // gitlab/runner version 11.5/all
	NodeIndex   string `env:"CI_NODE_INDEX"`     // gitlab/runner version 11.5/all

	// Repo Info
	ProjectPath string `env:"CI_PROJECT_PATH"` // gitlab/runner version 8.10/0.5
	ProjectURL  string `env:"CI_PROJECT_URL"`  // gitlab/runner version 8.10/0.5

	// Commit Info
	CommitSHA     string `env:"CI_COMMIT_SHA"`     // gitlab/runner version 9.0/all
	CommitAuthor  string `env:"CI_COMMIT_AUTHOR"`  // gitlab/runner version 13.11/all
	CommitBranch  string `env:"CI_COMMIT_BRANCH"`  // gitlab/runner version 12.6/0.5
	CommitMessage string `env:"CI_COMMIT_MESSAGE"` // gitlab/runner version 10.8/all

	// Consider in the future checking CI_SERVER_VERSION >= 13.2 (the newest version with all of these fields)
	// gitlab/runner version 11.7/all
	APIV4URL string `env:"CI_API_V4_URL"`
}

func (cfg GitLabEnv) MakeProvider() (Provider, error) {
	attemptedBy := cfg.UserLogin
	if attemptedBy == "" {
		// presumably if there's no attempted by, the build was triggered by pushing the commit / the commit author
		attemptedBy = cfg.CommitAuthor
	}

	tags, validationError := gitlabciTags(cfg)
	if validationError != nil {
		return Provider{}, validationError
	}

	provider := Provider{
		AttemptedBy:   attemptedBy,
		BranchName:    cfg.CommitBranch,
		CommitMessage: cfg.CommitMessage,
		CommitSha:     cfg.CommitSHA,
		JobTags:       tags,
		ProviderName:  "gitlabci",
		Title:         strings.Split(cfg.CommitMessage, "\n")[0],
	}

	return provider, nil
}

func gitlabciTags(cfg GitLabEnv) (map[string]any, error) {
	err := func() error {
		if cfg.JobName == "" {
			return errors.NewConfigurationError(
				"Missing job name",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your job name.",
				"You can configure GitLab's job name by setting the CI_JOB_NAME environment variable.",
			)
		}

		if cfg.JobStage == "" {
			return errors.NewConfigurationError(
				"Missing job stage",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your job stage.",
				"You can configure GitLab's job stage by setting the CI_JOB_STAGE environment variable.",
			)
		}

		if cfg.JobID == "" {
			return errors.NewConfigurationError(
				"Missing job ID",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your job ID.",
				"You can configure GitLab's job ID by setting the CI_JOB_ID environment variable.",
			)
		}

		if cfg.PipelineID == "" {
			return errors.NewConfigurationError(
				"Missing pipeline ID",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your pipeline ID.",
				"You can configure GitLab's pipeline ID by setting the CI_PIPELINE_ID environment variable.",
			)
		}

		if cfg.JobURL == "" {
			return errors.NewConfigurationError(
				"Missing job URL",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your job URL.",
				"You can configure GitLab's job URL by setting the CI_JOB_URL environment variable.",
			)
		}

		if cfg.PipelineURL == "" {
			return errors.NewConfigurationError(
				"Missing pipeline URL",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your pipeline URL.",
				"You can configure GitLab's pipeline URL by setting the CI_PIPELINE_URL environment variable.",
			)
		}

		if cfg.NodeTotal == "" {
			return errors.NewConfigurationError(
				"Missing node total",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your total node count.",
				"You can configure GitLab's node count by setting the CI_NODE_TOTAL environment variable.",
			)
		}

		if cfg.ProjectPath == "" {
			return errors.NewConfigurationError(
				"Missing project path",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your project path.",
				"You can configure GitLab's project path by setting the CI_PROJECT_PATH environment variable.",
			)
		}

		if cfg.ProjectURL == "" {
			return errors.NewConfigurationError(
				"Missing project URL",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine your project URL.",
				"You can configure GitLab's project URL by setting the CI_PROJECT_URL environment variable.",
			)
		}

		if cfg.APIV4URL == "" {
			return errors.NewConfigurationError(
				"Missing API URL",
				"It appears that you are running on a GitLab runner, however Captain is unable to determine GitLab's API endpoint.",
				"You can configure the API endpoint by setting the CI_API_V4_URL environment variable.",
			)
		}

		return nil
	}()
	if err != nil {
		return nil, err
	}
	// these get sent to captain as-is
	// name them to match the ENV vars from GitLab ci
	// so the names in captain have parallel meaning in the GitLab docs
	// (that's also the convention in other providers)
	tags := map[string]any{
		"gitlab_job_name":        cfg.JobName,
		"gitlab_job_stage":       cfg.JobStage,
		"gitlab_job_id":          cfg.JobID,
		"gitlab_pipeline_id":     cfg.PipelineID,
		"gitlab_job_url":         cfg.JobURL,
		"gitlab_pipeline_url":    cfg.PipelineURL,
		"gitlab_repository_path": cfg.ProjectPath,
		"gitlab_project_url":     cfg.ProjectURL,
		"gitlab_api_v4_url":      cfg.APIV4URL,
	}

	if cfg.NodeIndex != "" && cfg.NodeTotal != "" {
		tags["gitlab_node_index"] = cfg.NodeIndex
		tags["gitlab_node_total"] = cfg.NodeTotal
	}

	return tags, nil
}
