package providers

import (
	"fmt"
	"strconv"

	"github.com/rwx-research/captain-cli/internal/config"
	"github.com/rwx-research/captain-cli/internal/errors"
)

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

func (cfg CircleCIEnv) makeProvider() (Provider, error) {
	tags, validationError := circleciTags(cfg)

	if validationError != nil {
		return Provider{}, validationError
	}

	title := fmt.Sprintf("%s (%s)", cfg.Job, cfg.BuildNum)

	index, err := strconv.Atoi(cfg.NodeIndex)
	if err != nil {
		index = -1
	}

	total, err := strconv.Atoi(cfg.NodeTotal)
	if err != nil {
		total = -1
	}

	provider := Provider{
		AttemptedBy:   cfg.Username,
		BranchName:    cfg.Branch,
		CommitMessage: "",
		CommitSha:     cfg.Sha1,
		JobTags:       tags,
		ProviderName:  "circleci",
		Title:         title,
		PartitionNodes: config.PartitionNodes{
			Index: index,
			Total: total,
		},
	}

	return provider, nil
}

func circleciTags(cfg CircleCIEnv) (map[string]any, error) {
	err := func() error {
		// don't validate
		// ParallelJobIndex -- only present if parallelization enabled
		// ParallelJobTotal -- only present if parallelization enabled

		if cfg.BuildNum == "" {
			return errors.NewConfigurationError(
				"Missing build number",
				"It appears that you are running on CircleCI, however Captain is unable to determine your build number.",
				"You can configure CircleCI's build number by setting the CIRCLE_BUILD_NUM environment variable.",
			)
		}

		if cfg.BuildURL == "" {
			return errors.NewConfigurationError(
				"Missing build URL",
				"It appears that you are running on CircleCI, however Captain is unable to determine your build URL.",
				"You can configure CircleCI's build URL by setting the CIRCLE_BUILD_URL environment variable.",
			)
		}

		if cfg.Job == "" {
			return errors.NewConfigurationError(
				"Missing job name",
				"It appears that you are running on CircleCI, however Captain is unable to determine your job name.",
				"You can configure CircleCI's job name by setting the CIRCLE_JOB environment variable.",
			)
		}

		if cfg.ProjectUsername == "" {
			return errors.NewConfigurationError(
				"Missing project username",
				"It appears that you are running on CircleCI, however Captain is unable to determine your user-name.",
				"You can configure CircleCI's user-name by setting the CIRCLE_PROJECT_USERNAME environment variable.",
			)
		}

		if cfg.ProjectReponame == "" {
			return errors.NewConfigurationError(
				"Missing repository name",
				"It appears that you are running on CircleCI, however Captain is unable to determine your repository name.",
				"You can configure the repository name by setting the CIRCLE_PROJECT_REPONAME environment variable.",
			)
		}

		if cfg.RepositoryURL == "" {
			return errors.NewConfigurationError(
				"Missing repository URL",
				"It appears that you are running on CircleCI, however Captain is unable to determine your repository URL.",
				"You can configure the repository URL by setting the CIRCLE_REPOSITORY_URL environment variable.",
			)
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
