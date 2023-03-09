package providers

import "github.com/rwx-research/captain-cli/internal/errors"

type GitLabCIProvider struct {
	// build info
	JobName     string
	JobStage    string
	JobID       string
	PipelineID  string
	JobURL      string
	PipelineURL string
	AttemptedBy string
	NodeIndex   string // only present if parallelization enabled
	NodeTotal   string // always present

	// repository info
	RepoPath   string
	ProjectURL string

	// commit info
	CommitSHA     string
	CommitAuthor  string
	BranchName    string
	CommitMessage string

	// other info
	APIURL string
}

// a struct that mirrors env vars, see test/.env.GitLab
type GitLabCIEnv struct {
	// build info
	CiJobName       string
	CiJobStage      string
	CiJobID         string
	CiPipelineID    string
	CiJobURL        string
	CiPipelineURL   string
	GitlabUserLogin string
	CiNodeTotal     string
	CiNodeIndex     string

	// Repo Info
	CiProjectPath string
	CiProjectURL  string

	// Commit Info
	CiCommitSHA     string
	CiCommitAuthor  string
	CiCommitBranch  string
	CiCommitMessage string

	// other info
	CiAPIV4URL string
}

func (b GitLabCIProvider) GetAttemptedBy() string {
	if b.AttemptedBy != "" {
		return b.AttemptedBy
	}
	// presumably if there's no attempted by, the build was triggered by pushing the commit / the commit author
	return b.CommitAuthor
}

func (b GitLabCIProvider) GetBranchName() string {
	return b.BranchName // TODO: should this be RefName? What about tags?
}

func (b GitLabCIProvider) GetCommitSha() string {
	return b.CommitSHA
}

// not provided by GitLab ci. We'll reconstruct it on captain's side
func (b GitLabCIProvider) GetCommitMessage() string {
	return b.CommitMessage
}

func (b GitLabCIProvider) GetJobTags() map[string]any {
	// these get sent to captain as-is
	// name them to match the ENV vars from GitLab ci
	// so the names in captain have parallel meaning in the GitLab docs
	// (that's also the convention in other providers)
	tags := map[string]any{
		"gitlab_job_name":        b.JobName,
		"gitlab_job_stage":       b.JobStage,
		"gitlab_job_id":          b.JobID,
		"gitlab_pipeline_id":     b.PipelineID,
		"gitlab_job_url":         b.JobURL,
		"gitlab_pipeline_url":    b.PipelineURL,
		"gitlab_repository_path": b.RepoPath,
		"gitlab_project_url":     b.ProjectURL,
		"gitlab_api_v4_url":      b.APIURL,
	}

	if b.NodeIndex != "" && b.NodeTotal != "" {
		tags["gitlab_node_index"] = b.NodeIndex
		tags["gitlab_node_total"] = b.NodeTotal
	}
	return tags
}

func (b GitLabCIProvider) GetProviderName() string {
	return "gitlabci"
}

func (b GitLabCIProvider) Validate() error {
	// don't validate node index (it may be missing)
	// don't validate gitlab user login / attemptedBy (presumably it's missing if the build was triggered by pushing the commit from a non-gitlab account?)
	// TODO: check what is present when CI provider isn't gitlab

	if b.JobName == "" {
		return errors.NewConfigurationError("missing JobName")
	}

	if b.JobStage == "" {
		return errors.NewConfigurationError("missing JobStage")
	}

	if b.JobID == "" {
		return errors.NewConfigurationError("missing JobID")
	}

	if b.PipelineID == "" {
		return errors.NewConfigurationError("missing PipelineID")
	}

	if b.JobURL == "" {
		return errors.NewConfigurationError("missing JobURL")
	}

	if b.PipelineURL == "" {
		return errors.NewConfigurationError("missing PipelineURL")
	}

	if b.NodeTotal == "" {
		return errors.NewConfigurationError("missing NodeTotal")
	}

	if b.RepoPath == "" {
		return errors.NewConfigurationError("missing RepoPath")
	}

	if b.ProjectURL == "" {
		return errors.NewConfigurationError("missing ProjectURL")
	}

	if b.CommitSHA == "" {
		return errors.NewConfigurationError("missing CommitSHA")
	}

	if b.CommitAuthor == "" {
		return errors.NewConfigurationError("missing CommitAuthor")
	}

	if b.BranchName == "" {
		return errors.NewConfigurationError("missing BranchName")
	}

	if b.CommitMessage == "" {
		return errors.NewConfigurationError("missing CommitMessage")
	}

	if b.APIURL == "" {
		return errors.NewConfigurationError("missing APIURL")
	}

	return nil
}
