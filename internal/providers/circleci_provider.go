package providers

import "github.com/rwx-research/captain-cli/internal/errors"

type CircleciProvider struct {
	// build info
	BuildNum         string
	BuildURL         string
	JobName          string
	ParallelJobIndex string // only present if parallelization enabled
	ParallelJobTotal string // only present if parallelization enabled

	// repository info
	RepoAccountName string
	RepoName        string
	RepoURL         string

	// commit info
	BranchName  string
	CommitSha   string
	AttemptedBy string
}

// a struct that mirrors env vars, see test/.env.circleci
type CircleciEnv struct {
	CircleBuildNum        string
	CircleBuildURL        string
	CircleJob             string
	CircleNodeIndex       string
	CircleNodeTotal       string
	CircleProjectReponame string
	CircleProjectUsername string
	CircleRepositoryURL   string
	CircleBranch          string
	CircleSha1            string
	CircleUsername        string
}

func (b CircleciProvider) GetAttemptedBy() string {
	return b.AttemptedBy
}

func (b CircleciProvider) GetBranchName() string {
	return b.BranchName
}

func (b CircleciProvider) GetCommitSha() string {
	return b.CommitSha
}

// not provided by circle ci. We'll reconstruct it on captain's side
func (b CircleciProvider) GetCommitMessage() string {
	return ""
}

func (b CircleciProvider) GetJobTags() map[string]any {
	// these get sent to captain as-is
	// name them to match the ENV vars from circle ci
	// so the names in captain have parallel meaning in the circle docs
	// (that's also the convention in other providers)
	tags := map[string]any{
		"circle_build_num":        b.BuildNum,
		"circle_build_url":        b.BuildURL,
		"circle_job":              b.JobName,
		"circle_repository_url":   b.RepoURL,
		"circle_project_username": b.RepoAccountName,
		"circle_project_reponame": b.RepoName,
	}
	if b.ParallelJobIndex != "" && b.ParallelJobTotal != "" {
		tags["circle_node_index"] = b.ParallelJobIndex
		tags["circle_node_total"] = b.ParallelJobTotal
	}
	return tags
}

func (b CircleciProvider) GetProviderName() string {
	return "circleci"
}

func (b CircleciProvider) Validate() error {
	// don't validate
	// ParallelJobIndex -- only present if parallelization enabled
	// ParallelJobTotal -- only present if parallelization enabled

	if b.BuildNum == "" {
		return errors.NewConfigurationError("missing BuildNum")
	}

	if b.BuildURL == "" {
		return errors.NewConfigurationError("missing BuildURL")
	}

	if b.JobName == "" {
		return errors.NewConfigurationError("missing JobName")
	}

	if b.RepoAccountName == "" {
		return errors.NewConfigurationError("missing RepoAccountName")
	}

	if b.RepoName == "" {
		return errors.NewConfigurationError("missing RepoName")
	}

	if b.RepoURL == "" {
		return errors.NewConfigurationError("missing RepoURL")
	}

	if b.BranchName == "" {
		return errors.NewConfigurationError("missing BranchName")
	}

	if b.CommitSha == "" {
		return errors.NewConfigurationError("missing CommitSha")
	}

	if b.AttemptedBy == "" {
		return errors.NewConfigurationError("missing AttemptedBy")
	}

	return nil
}
