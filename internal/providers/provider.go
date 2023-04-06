package providers

import (
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
)

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
	Title         string
}

func Validate(p Provider) error {
	err := p.validate()
	if err == nil {
		return nil
	}

	if p.ProviderName == "generic" {
		return errors.NewConfigurationError(
			"Unsupported runtime environment",
			"Captain was unable to detect the presence of a supported CI provider.",
			"To use Captain locally or on unsupported hosts, please use the --who, --branch, and --sha flags. "+
				"Alternatively, these options can also be set via the CAPTAIN_WHO, CAPTAIN_BRANCH, and CAPTAIN_SHA "+
				"environment variables.",
		)
	}
	return err
}

// assigns default validation error if it exists
func (p Provider) validate() error {
	if p.ProviderName == "" {
		return errors.NewInternalError("provider name should never be empty")
	}

	if p.AttemptedBy == "" {
		return errors.NewConfigurationError(
			"Missing 'who'",
			"Captain requires the name of the user that triggered a build / test run.",
			"You can specify this name by using the --who flag. Alternatively you can also set this option "+
				"using the CAPTAIN_WHO environment variable.",
		)
	}

	if p.BranchName == "" {
		return errors.NewConfigurationError(
			"Missing branch name",
			"Captain requires a branch name in order to track test runs correctly.",
			"You can specify this name by using the --branch flag. Alternatively you can also set this option "+
				"using the CAPTAIN_BRANCH environment variable.",
		)
	}

	if p.CommitSha == "" {
		return errors.NewConfigurationError(
			"Missing commit SHA",
			"Captain requires a commit SHA in order to track test runs correctly.",
			"You can specify the SHA by using the --sha flag or the CAPTAIN_SHA environment variable",
		)
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
		into.Title = firstNonempty(from.Title, into.Title)
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

func (env Env) MakeProvider() (Provider, error) {
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

	mergedWithGeneric := Merge(detectedProvider, env.Generic.MakeProvider())

	if mergedWithGeneric.Title == "" {
		mergedWithGeneric.Title = "asdfjasdfj"
		mergedWithGeneric.Title = strings.Split(mergedWithGeneric.CommitMessage, "\n")[0]
	}

	// merge the generic provider into the detected provider. The captain-specific flags & env vars take precedence.
	return mergedWithGeneric, nil
}
