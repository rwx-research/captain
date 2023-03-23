package providers

import "github.com/rwx-research/captain-cli/internal/errors"

type CircleCIEnv struct {
	Detected bool `env:"CIRCLECI"`

	//	AttemptedBy
	Username string `env:"CIRCLE_USERNAME"`
	// branch
	Branch string `env:"CIRCLE_BRANCH"`
	// commit sha
	Sha1 string `env:"CIRCLE_SHA1"`
	// note: no commit message
	// tags
	BuildNum        string `env:"CIRCLE_BUILD_NUM"`
	BuildURL        string `env:"CIRCLE_BUILD_URL"`
	Job             string `env:"CIRCLE_JOB"`
	NodeIndex       string `env:"CIRCLE_NODE_INDEX"`
	NodeTotal       string `env:"CIRCLE_NODE_TOTAL"`
	ProjectReponame string `env:"CIRCLE_PROJECT_REPONAME"`
	ProjectUsername string `env:"CIRCLE_PROJECT_USERNAME"`
	RepositoryURL   string `env:"CIRCLE_REPOSITORY_URL"`
}

func (cfg CircleCIEnv) MakeProvider() (Provider, error) {
	tags, validationError := circleciTags(cfg)

	if validationError != nil {
		return Provider{}, validationError
	}

	provider := Provider{
		AttemptedBy:   cfg.Username,
		BranchName:    cfg.Branch,
		CommitMessage: "",
		CommitSha:     cfg.Sha1,
		JobTags:       tags,
		ProviderName:  "circleci",
	}

	return provider, nil
}

func circleciTags(cfg CircleCIEnv) (map[string]any, error) {
	err := func() error {
		// don't validate
		// ParallelJobIndex -- only present if parallelization enabled
		// ParallelJobTotal -- only present if parallelization enabled

		if cfg.BuildNum == "" {
			return errors.NewConfigurationError("missing Build Num")
		}

		if cfg.BuildURL == "" {
			return errors.NewConfigurationError("missing build URL")
		}

		if cfg.Job == "" {
			return errors.NewConfigurationError("missing job name")
		}

		if cfg.ProjectUsername == "" {
			return errors.NewConfigurationError("missing project username")
		}

		if cfg.ProjectReponame == "" {
			return errors.NewConfigurationError("missing project reponame")
		}

		if cfg.RepositoryURL == "" {
			return errors.NewConfigurationError("missing repository URL")
		}

		return nil
	}()
	// these get sent to captain as-is
	// name them to match the ENV vars from circle ci
	// so the names in captain have parallel meaning in the circle docs
	// (that's also the convention in other providers)
	tags := map[string]any{
		"circle_build_num":        cfg.BuildNum,
		"circle_build_url":        cfg.BuildURL,
		"circle_job":              cfg.Job,
		"circle_repository_url":   cfg.RepositoryURL,
		"circle_project_username": cfg.ProjectUsername,
		"circle_project_reponame": cfg.ProjectReponame,
	}
	if cfg.NodeIndex != "" && cfg.NodeTotal != "" {
		tags["circle_node_index"] = cfg.NodeIndex
		tags["circle_node_total"] = cfg.NodeTotal
	}
	return tags, err
}
