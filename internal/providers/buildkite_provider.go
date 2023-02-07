package providers

import "github.com/rwx-research/captain-cli/internal/errors"

type BuildkiteEnv struct {
	BuildkiteBranch            string
	BuildkiteBuildCreatorEmail string
	BuildkiteBuildID           string
	BuildkiteBuildURL          string
	BuildkiteCommit            string
	BuildkiteJobID             string
	BuildkiteLabel             string
	BuildkiteMessage           string
	BuildkiteOrganizationSlug  string
	BuildkiteParallelJob       string
	BuildkiteParallelJobCount  string
	BuildkiteRepo              string
	BuildkiteRetryCount        string
}
type BuildkiteProvider struct {
	Env           BuildkiteEnv
	GitRepository struct {
		Branch        string
		CommitSha     string
		CommitMessage string
	}
}

func (b BuildkiteProvider) GetAttemptedBy() string {
	return b.Env.BuildkiteBuildCreatorEmail
}

func (b BuildkiteProvider) GetBranchName() string {
	if b.GitRepository.Branch != "" {
		return b.GitRepository.Branch
	}

	return b.Env.BuildkiteBranch
}

func (b BuildkiteProvider) GetCommitSha() string {
	if b.GitRepository.CommitSha != "" {
		return b.GitRepository.CommitSha
	}

	return b.Env.BuildkiteCommit
}

func (b BuildkiteProvider) GetCommitMessage() string {
	if b.GitRepository.CommitMessage != "" {
		return b.GitRepository.CommitMessage
	}

	return b.Env.BuildkiteMessage
}

func (b BuildkiteProvider) GetJobTags() map[string]any {
	tags := map[string]any{
		"buildkite_build_id":          b.Env.BuildkiteBuildID,
		"buildkite_build_url":         b.Env.BuildkiteBuildURL,
		"buildkite_retry_count":       b.Env.BuildkiteRetryCount,
		"buildkite_job_id":            b.Env.BuildkiteJobID,
		"buildkite_label":             b.Env.BuildkiteLabel,
		"buildkite_repo":              b.Env.BuildkiteRepo,
		"buildkite_organization_slug": b.Env.BuildkiteOrganizationSlug,
	}
	if b.Env.BuildkiteParallelJob != "" {
		tags["buildkite_parallel_job"] = b.Env.BuildkiteParallelJob
	}
	if b.Env.BuildkiteParallelJobCount != "" {
		tags["buildkite_parallel_job_count"] = b.Env.BuildkiteParallelJobCount
	}
	return tags
}

func (b BuildkiteProvider) GetProviderName() string {
	return "buildkite"
}

func (b BuildkiteProvider) Validate() error {
	if b.Env.BuildkiteBuildCreatorEmail == "" {
		return errors.NewConfigurationError("missing BuildkiteBuildCreatorEmail")
	}

	if b.Env.BuildkiteOrganizationSlug == "" {
		return errors.NewConfigurationError("missing BuildkiteOrganizationSlug")
	}

	if b.Env.BuildkiteRepo == "" {
		return errors.NewConfigurationError("missing BuildkiteRepo")
	}

	if b.Env.BuildkiteLabel == "" {
		return errors.NewConfigurationError("missing BuildkiteLabel")
	}

	if b.Env.BuildkiteJobID == "" {
		return errors.NewConfigurationError("missing BuildkiteJobID")
	}

	if b.Env.BuildkiteRetryCount == "" {
		return errors.NewConfigurationError("missing BuildkiteRetryCount")
	}

	if b.Env.BuildkiteBuildID == "" {
		return errors.NewConfigurationError("missing BuildkiteBuildID")
	}

	if b.Env.BuildkiteBuildURL == "" {
		return errors.NewConfigurationError("missing BuildkiteBuildURL")
	}

	if b.Env.BuildkiteBranch == "" {
		return errors.NewConfigurationError("missing BuildkiteBranch")
	}

	if b.Env.BuildkiteCommit == "" {
		return errors.NewConfigurationError("missing BuildkiteCommit")
	}

	return nil
}
