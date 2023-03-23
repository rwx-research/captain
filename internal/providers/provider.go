package providers

import "github.com/rwx-research/captain-cli/internal/errors"

type Env struct {
	Buildkite BuildkiteEnv
	CircleCI  CircleCIEnv
	Generic   GenericEnv
	GitHub    GitHubEnv
	GitLab    GitLabEnv
}

type Provider struct {
	AttemptedBy   string
	BranchName    string
	CommitMessage string
	CommitSha     string
	JobTags       map[string]any
	ProviderName  string
}

func Validate(p Provider) error {
	err := p.validate()
	if err == nil {
		return nil
	}

	if p.ProviderName == "generic" {
		return errors.NewConfigurationError("Could not detect a supported CI provider. Without a " +
			"supported CI provider, we require --who, --branch, and --sha to be set.")
	}
	return err
}

// assigns default validation error if it exists
func (p Provider) validate() error {
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

func (env Env) MakeProviderAdapter() (Provider, error) {
	// detect provider from environment if we can
	wrapError := func(p Provider, err error) (Provider, error) {
		return p, errors.Wrap(err, "error building detected provider")
	}
	detectedProvider, err := func() (Provider, error) {
		switch {
		case env.GitHub.Detected:
			return wrapError(env.GitHub.MakeProvider())
		case env.Buildkite.Detected:
			return wrapError(env.Buildkite.MakeProvider())
		case env.CircleCI.Detected:
			return wrapError(env.CircleCI.MakeProvider())
		case env.GitLab.Detected:
			return wrapError(env.GitLab.MakeProvider())
		}
		return Provider{}, nil
	}()
	if err != nil {
		return Provider{}, err
	}

	// merge the generic provider into the detected provider. The captain-specific flags & env vars take precedence.
	return Merge(detectedProvider, env.Generic.MakeProvider()), nil
}
