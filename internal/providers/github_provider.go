package providers

import (
	"encoding/json"

	"github.com/rwx-research/captain-cli/internal/errors"
)

type GithubProvider struct {
	AccountName    string
	AttemptedBy    string
	BranchName     string
	CommitMessage  string
	CommitSha      string
	JobName        string
	JobMatrix      string
	RunAttempt     string
	RunID          string
	RepositoryName string
}

func (g GithubProvider) GetAttemptedBy() string {
	return g.AttemptedBy
}

func (g GithubProvider) GetBranchName() string {
	return g.BranchName
}

func (g GithubProvider) GetCommitSha() string {
	return g.CommitSha
}

func (g GithubProvider) GetCommitMessage() string {
	return g.CommitMessage
}

func (g GithubProvider) GetJobTags() map[string]any {
	tags := map[string]any{
		"github_run_id":          g.RunID,
		"github_run_attempt":     g.RunAttempt,
		"github_repository_name": g.RepositoryName,
		"github_account_owner":   g.AccountName,
		"github_job_name":        g.JobName,
	}
	if g.JobMatrix != "" {
		rawJobMatrix := json.RawMessage(g.JobMatrix)
		tags["github_job_matrix"] = &rawJobMatrix
	}
	return tags
}

func (g GithubProvider) GetProviderName() string {
	return "github"
}

func (g GithubProvider) Validate() error {
	if g.AttemptedBy == "" {
		return errors.NewConfigurationError("missing attempted by")
	}

	if g.AccountName == "" {
		return errors.NewConfigurationError("missing account name")
	}

	if g.JobName == "" {
		return errors.NewConfigurationError("missing job name")
	}

	if g.RunAttempt == "" {
		return errors.NewConfigurationError("missing run attempt")
	}

	if g.RunID == "" {
		return errors.NewConfigurationError("missing run ID")
	}

	if g.RepositoryName == "" {
		return errors.NewConfigurationError("missing repository name")
	}

	if g.BranchName == "" {
		return errors.NewConfigurationError("missing branch name")
	}

	if g.CommitSha == "" {
		return errors.NewConfigurationError("missing commit sha")
	}

	if g.JobMatrix != "" && g.JobMatrix != "null" {
		if !json.Valid([]byte(g.JobMatrix)) || g.JobMatrix[0] != byte('{') {
			return errors.NewConfigurationError(
				`invalid github-job-matrix value supplied.
Please provide '${{ toJSON(matrix) }}' with single quotes.
This information is required due to a limitation with GitHub.`)
		}
	}

	return nil
}
