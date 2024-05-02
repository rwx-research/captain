package providers

import (
	"strconv"

	"github.com/rwx-research/captain-cli/internal/config"
	"github.com/rwx-research/captain-cli/internal/errors"
)

type MintEnv struct {
	Detected bool `env:"MINT"`

	ParallelIndex     *string `env:"MINT_PARALLEL_INDEX"`
	ParallelTotal     *string `env:"MINT_PARALLEL_TOTAL"`
	Actor             string  `env:"MINT_ACTOR"`
	ActorID           *string `env:"MINT_ACTOR_ID"`
	RunURL            string  `env:"MINT_RUN_URL"`
	TaskURL           string  `env:"MINT_TASK_URL"`
	RunID             string  `env:"MINT_RUN_ID"`
	TaskID            string  `env:"MINT_TASK_ID"`
	TaskAttemptNumber string  `env:"MINT_TASK_ATTEMPT_NUMBER"`
	RunTitle          string  `env:"MINT_RUN_TITLE"`
	GitRepositoryURL  string  `env:"MINT_GIT_REPOSITORY_URL"`
	GitRepositoryName string  `env:"MINT_GIT_REPOSITORY_NAME"`
	GitCommitMessage  string  `env:"MINT_GIT_COMMIT_MESSAGE"`
	GitCommitSha      string  `env:"MINT_GIT_COMMIT_SHA"`
	GitRef            string  `env:"MINT_GIT_REF"`
	GitRefName        string  `env:"MINT_GIT_REF_NAME"`
}

func (cfg MintEnv) makeProvider() (Provider, error) {
	tags, validationError := mintTags(cfg)

	if validationError != nil {
		return Provider{}, validationError
	}

	index := -1
	if cfg.ParallelIndex != nil {
		var err error
		index, err = strconv.Atoi(*cfg.ParallelIndex)
		if err != nil {
			index = -1
		}
	}

	total := -1
	if cfg.ParallelTotal != nil {
		var err error
		total, err = strconv.Atoi(*cfg.ParallelTotal)
		if err != nil {
			total = -1
		}
	}

	provider := Provider{
		AttemptedBy:   cfg.Actor,
		BranchName:    cfg.GitRefName,
		CommitMessage: cfg.GitCommitMessage,
		CommitSha:     cfg.GitCommitSha,
		JobTags:       tags,
		ProviderName:  "mint",
		Title:         cfg.RunTitle,
		PartitionNodes: config.PartitionNodes{
			Index: index,
			Total: total,
		},
	}

	return provider, nil
}

func mintTags(cfg MintEnv) (map[string]any, error) {
	err := func() error {
		// don't validate
		// ActorID -- expected to be sometimes unset
		// ParallelIndex -- only present if parallelization enabled
		// ParallelTotal -- only present if parallelization enabled

		if cfg.Actor == "" {
			return errors.NewConfigurationError(
				"Missing actor",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your actor.",
				"Make sure you are running on Mint. If not, set `MINT` to `false`.",
			)
		}

		if cfg.RunURL == "" {
			return errors.NewConfigurationError(
				"Missing run url",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your run url.",
				"Make sure you are running on Mint. If not, set `MINT` to `false`.",
			)
		}

		if cfg.TaskURL == "" {
			return errors.NewConfigurationError(
				"Missing task url",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your task url.",
				"Make sure you are running on Mint. If not, set `MINT` to `false`.",
			)
		}

		if cfg.RunID == "" {
			return errors.NewConfigurationError(
				"Missing run id",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your run id.",
				"Make sure you are running on Mint. If not, set `MINT` to `false`.",
			)
		}

		if cfg.TaskID == "" {
			return errors.NewConfigurationError(
				"Missing task id",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your task id.",
				"Make sure you are running on Mint. If not, set `MINT` to `false`.",
			)
		}

		if cfg.TaskAttemptNumber == "" {
			return errors.NewConfigurationError(
				"Missing task attempt number",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your task attempt number.",
				"Make sure you are running on Mint. If not, set `MINT` to `false`.",
			)
		}

		if cfg.RunTitle == "" {
			return errors.NewConfigurationError(
				"Missing run title",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your run title.",
				"Make sure you are running on Mint. If not, set `MINT` to `false`.",
			)
		}

		if cfg.GitRepositoryURL == "" {
			return errors.NewConfigurationError(
				"Missing git repository url",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your git repository url.",
				"Ensure that you've run the `mint/git-clone` leaf to set the `MINT_GIT_*` metadata.",
			)
		}

		if cfg.GitRepositoryName == "" {
			return errors.NewConfigurationError(
				"Missing git repository name",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your git repository name.",
				"Ensure that you've run the `mint/git-clone` leaf to set the `MINT_GIT_*` metadata.",
			)
		}

		if cfg.GitCommitMessage == "" {
			return errors.NewConfigurationError(
				"Missing git commit message",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your git commit message.",
				"Ensure that you've run the `mint/git-clone` leaf to set the `MINT_GIT_*` metadata.",
			)
		}

		if cfg.GitCommitSha == "" {
			return errors.NewConfigurationError(
				"Missing git commit sha",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your git commit sha.",
				"Ensure that you've run the `mint/git-clone` leaf to set the `MINT_GIT_*` metadata.",
			)
		}

		if cfg.GitRef == "" {
			return errors.NewConfigurationError(
				"Missing git ref",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your git ref.",
				"Ensure that you've run the `mint/git-clone` leaf to set the `MINT_GIT_*` metadata.",
			)
		}

		if cfg.GitRefName == "" {
			return errors.NewConfigurationError(
				"Missing git ref name",
				"It appears that you are running on Mint (`MINT` is set to `true`),"+
					" however Captain is unable to determine your git ref name.",
				"Ensure that you've run the `mint/git-clone` leaf to set the `MINT_GIT_*` metadata.",
			)
		}

		return nil
	}()

	tags := map[string]any{
		"mint_actor":               cfg.Actor,
		"mint_run_url":             cfg.RunURL,
		"mint_task_url":            cfg.TaskURL,
		"mint_run_id":              cfg.RunID,
		"mint_task_id":             cfg.TaskID,
		"mint_task_attempt_number": cfg.TaskAttemptNumber,
		"mint_run_title":           cfg.RunTitle,
		"mint_git_repository_url":  cfg.GitRepositoryURL,
		"mint_git_repository_name": cfg.GitRepositoryName,
		"mint_git_commit_message":  cfg.GitCommitMessage,
		"mint_git_commit_sha":      cfg.GitCommitSha,
		"mint_git_ref":             cfg.GitRef,
		"mint_git_ref_name":        cfg.GitRefName,
	}

	if cfg.ParallelIndex != nil {
		tags["mint_parallel_index"] = *cfg.ParallelIndex
	} else {
		tags["mint_parallel_index"] = nil
	}

	if cfg.ParallelTotal != nil {
		tags["mint_parallel_total"] = *cfg.ParallelTotal
	} else {
		tags["mint_parallel_total"] = nil
	}

	if cfg.ActorID != nil {
		tags["mint_actor_id"] = *cfg.ActorID
	} else {
		tags["mint_actor_id"] = nil
	}

	return tags, err
}
