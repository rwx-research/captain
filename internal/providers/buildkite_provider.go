package providers

import "github.com/rwx-research/captain-cli/internal/errors"

type BuildkiteProvider struct {
	Branch            string
	BuildCreatorEmail string
	BuildID           string
	BuildURL          string
	Commit            string
	JobID             string
	Label             string
	Message           string
	OrganizationSlug  string
	ParallelJob       string
	ParallelJobCount  string
	Repo              string
	RetryCount        string
}

func (b BuildkiteProvider) GetAttemptedBy() string {
	return b.BuildCreatorEmail
}

func (b BuildkiteProvider) GetBranchName() string {
	return b.Branch
}

func (b BuildkiteProvider) GetCommitSha() string {
	return b.Commit
}

func (b BuildkiteProvider) GetCommitMessage() string {
	return b.Message
}

func (b BuildkiteProvider) GetJobTags() map[string]any {
	tags := map[string]any{
		"buildkite_build_id":          b.BuildID,
		"buildkite_build_url":         b.BuildURL,
		"buildkite_retry_count":       b.RetryCount,
		"buildkite_job_id":            b.JobID,
		"buildkite_label":             b.Label,
		"buildkite_repo":              b.Repo,
		"buildkite_organization_slug": b.OrganizationSlug,
	}

	if b.ParallelJob != "" {
		tags["buildkite_parallel_job"] = b.ParallelJob
	}

	if b.ParallelJobCount != "" {
		tags["buildkite_parallel_job_count"] = b.ParallelJobCount
	}

	return tags
}

func (b BuildkiteProvider) GetProviderName() string {
	return "buildkite"
}

func (b BuildkiteProvider) Validate() error {
	if b.BuildCreatorEmail == "" {
		return errors.NewConfigurationError("missing BuildCreatorEmail")
	}

	if b.OrganizationSlug == "" {
		return errors.NewConfigurationError("missing OrganizationSlug")
	}

	if b.Repo == "" {
		return errors.NewConfigurationError("missing Repo")
	}

	if b.Label == "" {
		return errors.NewConfigurationError("missing Label")
	}

	if b.JobID == "" {
		return errors.NewConfigurationError("missing JobID")
	}

	if b.RetryCount == "" {
		return errors.NewConfigurationError("missing RetryCount")
	}

	if b.BuildID == "" {
		return errors.NewConfigurationError("missing BuildID")
	}

	if b.BuildURL == "" {
		return errors.NewConfigurationError("missing BuildURL")
	}

	if b.Branch == "" {
		return errors.NewConfigurationError("missing Branch")
	}

	if b.Commit == "" {
		return errors.NewConfigurationError("missing Commit")
	}

	return nil
}
