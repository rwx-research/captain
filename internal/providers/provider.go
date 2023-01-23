package providers

type Provider interface {
	Validate() error
	GetAttemptedBy() string
	GetBranchName() string
	GetCommitSha() string
	GetJobTags() map[string]any
	GetProviderName() string
	GetCommitMessage() string
}
