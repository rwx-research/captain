package providers

import (
	"strconv"

	"github.com/rwx-research/captain-cli/internal/config"
	"github.com/rwx-research/captain-cli/internal/errors"
)

type BuildkiteEnv struct {
	Detected bool `env:"BUILDKITE"`

	// AttemptedBy
	BuildCreatorEmail string `env:"BUILDKITE_BUILD_CREATOR_EMAIL"`
	// branch
	Branch string `env:"BUILDKITE_BRANCH"`
	// commit message
	Message string `env:"BUILDKITE_MESSAGE"`
	// commit sha
	Commit string `env:"BUILDKITE_COMMIT"`
	// tags
	BuildID          string `env:"BUILDKITE_BUILD_ID"`
	BuildURL         string `env:"BUILDKITE_BUILD_URL"`
	JobID            string `env:"BUILDKITE_JOB_ID"`
	Label            string `env:"BUILDKITE_LABEL"`
	OrganizationSlug string `env:"BUILDKITE_ORGANIZATION_SLUG"`
	ParallelJob      string `env:"BUILDKITE_PARALLEL_JOB"`
	ParallelJobCount string `env:"BUILDKITE_PARALLEL_JOB_COUNT"`
	Repo             string `env:"BUILDKITE_REPO"`
	RetryCount       string `env:"BUILDKITE_RETRY_COUNT"`
}

func (cfg BuildkiteEnv) makeProvider() (Provider, error) {
	tags, validationError := buildkiteTags(cfg)

	if validationError != nil {
		return Provider{}, validationError
	}

	index, err := strconv.Atoi(cfg.ParallelJob)
	if err != nil {
		index = -1
	}

	total, err := strconv.Atoi(cfg.ParallelJobCount)
	if err != nil {
		total = -1
	}

	provider := Provider{
		AttemptedBy:   cfg.BuildCreatorEmail,
		BranchName:    cfg.Branch,
		CommitMessage: cfg.Message,
		CommitSha:     cfg.Commit,
		JobTags:       tags,
		ProviderName:  "buildkite",
		PartitionNodes: config.PartitionNodes{
			Index: index,
			Total: total,
		},
	}

	return provider, nil
}

func buildkiteTags(cfg BuildkiteEnv) (map[string]any, error) {
	err := func() error {
		if cfg.OrganizationSlug == "" {
			return errors.NewConfigurationError(
				"Missing organization slug",
				"It appears that you are running on Buildkite, however Captain is unable to determine your organization slug.",
				"You can configure Buildkite's organization slug by setting the BUILDKITE_ORGANIZATION_SLUG environment variable.",
			)
		}

		if cfg.Repo == "" {
			return errors.NewConfigurationError(
				"Missing repository",
				"It appears that you are running on Buildkite, however Captain is unable to determine your repository name.",
				"You can configure Buildkite's repository name by setting the BUILDKITE_REPO environment variable.",
			)
		}

		if cfg.Label == "" {
			return errors.NewConfigurationError(
				"Missing label",
				"It appears that you are running on Buildkite, however Captain is unable to read Buildkite's labels.",
				"You can configure labels by setting the BUILDKITE_LABEL environment variable.",
			)
		}

		if cfg.JobID == "" {
			return errors.NewConfigurationError(
				"Missing job ID",
				"It appears that you are running on Buildkite, however Captain is unable to determine Buildkite's job ID.",
				"You can configure the job ID by setting the BUILDKITE_JOB_ID environment variable.",
			)
		}

		if cfg.RetryCount == "" {
			return errors.NewConfigurationError(
				"Missing retry count",
				"It appears that you are running on Buildkite, however Captain is unable to determine Buildkite's retry count.",
				"You can configure the retry count by setting the BUILDKITE_RETRY_COUNT environment variable.",
			)
		}

		if cfg.BuildID == "" {
			return errors.NewConfigurationError(
				"Missing build ID",
				"It appears that you are running on Buildkite, however Captain is unable to determine Buildkite's build ID.",
				"You can configure the build ID by setting the BUILDKITE_BUILD_ID environment variable.",
			)
		}

		if cfg.BuildURL == "" {
			return errors.NewConfigurationError(
				"Missing build URL",
				"It appears that you are running on Buildkite, however Captain is unable to determine Buildkite's build URL.",
				"You can configure the build URL by setting the BUILDKITE_BUILD_URL environment variable.",
			)
		}

		return nil
	}()

	tags := map[string]any{
		"buildkite_build_id":          cfg.BuildID,
		"buildkite_build_url":         cfg.BuildURL,
		"buildkite_retry_count":       cfg.RetryCount,
		"buildkite_job_id":            cfg.JobID,
		"buildkite_label":             cfg.Label,
		"buildkite_repo":              cfg.Repo,
		"buildkite_organization_slug": cfg.OrganizationSlug,
	}

	if cfg.ParallelJob != "" {
		tags["buildkite_parallel_job"] = cfg.ParallelJob
	}

	if cfg.ParallelJobCount != "" {
		tags["buildkite_parallel_job_count"] = cfg.ParallelJobCount
	}

	return tags, err
}
