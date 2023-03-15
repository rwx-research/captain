package providers

import "github.com/rwx-research/captain-cli/internal/errors"

type NullProvider struct{}

func (p NullProvider) GetAttemptedBy() string {
	return ""
}

func (p NullProvider) GetBranchName() string {
	return ""
}

func (p NullProvider) GetCommitSha() string {
	return ""
}

func (p NullProvider) GetCommitMessage() string {
	return ""
}

func (p NullProvider) GetJobTags() map[string]any {
	return map[string]any{}
}

func (p NullProvider) GetProviderName() string {
	return "unsupported"
}

func (p NullProvider) Validate() error {
	return errors.NewConfigurationError("We've failed to detect a supported CI context")
}
