package providers

import (
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

func MakeBuildkiteProvider(cfg BuildkiteEnv) (Provider, error) {
	tags, validationError := buildkiteTags(cfg)

	if validationError != nil {
		return Provider{}, validationError
	}

	provider := Provider{
		AttemptedBy:   cfg.BuildCreatorEmail,
		BranchName:    cfg.Branch,
		CommitMessage: cfg.Message,
		CommitSha:     cfg.Commit,
		JobTags:       tags,
		ProviderName:  "buildkite",
	}

	return provider, nil
}

func buildkiteTags(cfg BuildkiteEnv) (map[string]any, error) {
	err := func() error {
		if cfg.OrganizationSlug == "" {
			return errors.NewConfigurationError("missing OrganizationSlug")
		}

		if cfg.Repo == "" {
			return errors.NewConfigurationError("missing Repo")
		}

		if cfg.Label == "" {
			return errors.NewConfigurationError("missing Label")
		}

		if cfg.JobID == "" {
			return errors.NewConfigurationError("missing JobID")
		}

		if cfg.RetryCount == "" {
			return errors.NewConfigurationError("missing RetryCount")
		}

		if cfg.BuildID == "" {
			return errors.NewConfigurationError("missing BuildID")
		}

		if cfg.BuildURL == "" {
			return errors.NewConfigurationError("missing BuildURL")
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
