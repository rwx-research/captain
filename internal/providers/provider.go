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
func (p *Provider) Validate() error {
	if p == nil {
		return errors.NewInternalError("provider should never be nil")
	}

	if p.AttemptedBy == "" {
		return errors.NewConfigurationError("missing attempted by")
	}

	if p.BranchName == "" {
		return errors.NewConfigurationError("missing branch name")
	}

	if p.CommitMessage == "" && p.ProviderName != "circleci" && p.ProviderName != "generic" {
		return errors.NewConfigurationError("missing commit message")
	}

	if p.CommitSha == "" {
		return errors.NewConfigurationError("missing commit sha")
	}
	return nil
}

func Merge(into Provider, from Provider) Provider {
	if into.ProviderName != "" {
		into.AttemptedBy = mergeStrings(from.AttemptedBy, into.AttemptedBy)
		into.BranchName = mergeStrings(from.BranchName, into.BranchName)
		into.CommitSha = mergeStrings(from.CommitSha, into.CommitSha)
		return into
	}
	return from
}

// return the first non-empty string
func mergeStrings(strs ...string) string {
	for _, str := range strs {
		if str != "" {
			return str
		}
	}

	return ""
}
