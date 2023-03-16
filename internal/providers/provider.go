package providers

import "github.com/rwx-research/captain-cli/internal/errors"

type Provider struct {
	AttemptedBy   string
	BranchName    string
	CommitMessage string
	CommitSha     string
	JobTags       map[string]any
	ProviderName  string
}

// assigns default validation error if it exists
func (p Provider) Validate() error {
	if p.ProviderName == "" {
		return errors.NewInternalError("provider name should never be empty")
	}

	if p.AttemptedBy == "" {
		return errors.NewConfigurationError("missing attempted by")
	}

	if p.BranchName == "" {
		return errors.NewConfigurationError("missing branch name")
	}

	if p.CommitSha == "" {
		return errors.NewConfigurationError("missing commit sha")
	}

	return nil
}

// merge merges attempted by, branch name, and commitsha
// preferring the values from `from` if they are not empty
// if into has an empty provider name, we assume it's not set and take all values from `from`
func Merge(into Provider, from Provider) Provider {
	if into.ProviderName != "" {
		into.AttemptedBy = firstNonempty(from.AttemptedBy, into.AttemptedBy)
		into.BranchName = firstNonempty(from.BranchName, into.BranchName)
		into.CommitSha = firstNonempty(from.CommitSha, into.CommitSha)
		into.CommitMessage = firstNonempty(from.CommitMessage, into.CommitMessage)
		return into
	}
	return from
}

// return the first non-empty string
func firstNonempty(strs ...string) string {
	for _, str := range strs {
		if str != "" {
			return str
		}
	}

	return ""
}
