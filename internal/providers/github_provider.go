package providers

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
)

type GitHubEnv struct {
	Detected bool `env:"GITHUB_ACTIONS"`

	// attempted by
	ExecutingActor  string `env:"GITHUB_ACTOR"`
	TriggeringActor string `env:"GITHUB_TRIGGERING_ACTOR"`

	// branch
	EventName string `env:"GITHUB_EVENT_NAME"`
	RefName   string `env:"GITHUB_REF_NAME"`
	HeadRef   string `env:"GITHUB_HEAD_REF"`

	// commit message is parsed from the event payload
	EventPath string `env:"GITHUB_EVENT_PATH"`

	// commit sha
	CommitSha string `env:"GITHUB_SHA"`

	// tags
	Attempt    string `env:"GITHUB_RUN_ATTEMPT"`
	ID         string `env:"GITHUB_RUN_ID"`
	Name       string `env:"GITHUB_JOB"`
	Repository string `env:"GITHUB_REPOSITORY"`
}

func (cfg GitHubEnv) MakeProvider() (Provider, error) {
	eventPayloadData := struct {
		HeadCommit struct {
			Message string `json:"message"`
		} `json:"head_commit"`
	}{}

	file, err := os.Open(cfg.EventPath)
	if err != nil && !os.IsNotExist(err) {
		return Provider{}, errors.Wrap(err, "unable to open event payload file")
	} else if err == nil {
		if err := json.NewDecoder(file).Decode(&eventPayloadData); err != nil {
			return Provider{}, errors.Wrap(err, "failed to decode event payload data")
		}
	}

	return cfg.MakeProviderWithoutCommitMessageParsing(eventPayloadData.HeadCommit.Message)
}

// this function is only here to make the provider easier to test without writing the event file to disk
func (cfg GitHubEnv) MakeProviderWithoutCommitMessageParsing(message string) (Provider, error) {
	branchName := cfg.RefName
	if cfg.EventName == "pull_request" {
		branchName = cfg.HeadRef
	}

	attemptedBy := cfg.TriggeringActor
	if attemptedBy == "" {
		attemptedBy = cfg.ExecutingActor
	}

	tags, validationError := githubTags(cfg)
	if validationError != nil {
		return Provider{}, validationError
	}

	provider := Provider{
		AttemptedBy:   attemptedBy,
		BranchName:    branchName,
		CommitMessage: message,
		CommitSha:     cfg.CommitSha,
		JobTags:       tags,
		ProviderName:  "github",
	}

	return provider, nil
}

func githubTags(cfg GitHubEnv) (map[string]any, error) {
	err := func() error {
		if cfg.ID == "" {
			return errors.NewConfigurationError("missing run ID")
		}

		if cfg.Attempt == "" {
			return errors.NewConfigurationError("missing run attempt")
		}

		if cfg.Name == "" {
			return errors.NewConfigurationError("missing job name")
		}

		if cfg.Repository == "" {
			return errors.NewConfigurationError("missing repository")
		}

		return nil
	}()
	owner, repo := path.Split(cfg.Repository)

	tags := map[string]any{
		"github_run_id":          cfg.ID,
		"github_run_attempt":     cfg.Attempt,
		"github_repository_name": repo,
		"github_account_owner":   strings.TrimSuffix(owner, "/"),
		"github_job_name":        cfg.Name,
	}

	return tags, err
}
