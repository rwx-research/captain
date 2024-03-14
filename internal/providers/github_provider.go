package providers

import (
	"encoding/json"
	"fmt"
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

	// workflow
	Workflow string `env:"GITHUB_WORKFLOW"`

	// tags
	Attempt    string `env:"GITHUB_RUN_ATTEMPT"`
	ID         string `env:"GITHUB_RUN_ID"`
	Name       string `env:"GITHUB_JOB"`
	Repository string `env:"GITHUB_REPOSITORY"`
}

type GitHubEventPayloadData struct {
	HeadCommit struct {
		Message string `json:"message"`
	} `json:"head_commit"`
	PullRequest struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	} `json:"pull_request"`
	Issue struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	} `json:"issue"`
}

func (cfg GitHubEnv) makeProvider() (Provider, error) {
	eventPayloadData := GitHubEventPayloadData{}

	file, err := os.Open(cfg.EventPath)
	if err != nil && !os.IsNotExist(err) {
		return Provider{}, errors.Wrap(err, "unable to open event payload file")
	} else if err == nil {
		if err := json.NewDecoder(file).Decode(&eventPayloadData); err != nil {
			return Provider{}, errors.Wrap(err, "failed to decode event payload data")
		}
	}

	return cfg.MakeProviderWithoutCommitMessageParsing(eventPayloadData)
}

// this function is only here to make the provider easier to test without writing the event file to disk
func (cfg GitHubEnv) MakeProviderWithoutCommitMessageParsing(payloadData GitHubEventPayloadData) (Provider, error) {
	branchName := cfg.RefName
	if cfg.EventName == "pull_request" {
		branchName = cfg.HeadRef
	}

	attemptedBy := cfg.TriggeringActor
	if attemptedBy == "" {
		attemptedBy = cfg.ExecutingActor
	}

	tags, validationError := githubTags(cfg, payloadData)
	if validationError != nil {
		return Provider{}, validationError
	}

	if payloadData.PullRequest.Title == "" && payloadData.Issue.Title != "" {
		payloadData.PullRequest.Title = payloadData.Issue.Title
		payloadData.PullRequest.Number = payloadData.Issue.Number
	}

	commitMessage := payloadData.HeadCommit.Message
	var title string
	if commitMessage == "" {
		if payloadData.PullRequest.Title == "" {
			title = cfg.Workflow
		} else {
			title = fmt.Sprintf("%s (PR #%d)", payloadData.PullRequest.Title, payloadData.PullRequest.Number)
		}
	}

	provider := Provider{
		AttemptedBy:   attemptedBy,
		BranchName:    branchName,
		CommitMessage: commitMessage,
		CommitSha:     cfg.CommitSha,
		JobTags:       tags,
		ProviderName:  "github",
		Title:         title,
	}

	return provider, nil
}

func githubTags(cfg GitHubEnv, eventPayloadData GitHubEventPayloadData) (map[string]any, error) {
	err := func() error {
		if cfg.ID == "" {
			return errors.NewConfigurationError(
				"Missing run ID",
				"It appears that you are running on Github Actions, however Captain is unable to determine your run ID.",
				"You can configure Github's run ID by setting the GITHUB_RUN_ID environment variable.",
			)
		}

		if cfg.Attempt == "" {
			return errors.NewConfigurationError(
				"Missing run attempt",
				"It appears that you are running on Github Actions, however Captain is unable to determine your run attempt "+
					"number.",
				"You can configure Github's run attempt number by setting the GITHUB_RUN_ATTEMPT environment variable.",
			)
		}

		if cfg.Name == "" {
			return errors.NewConfigurationError(
				"Missing job name",
				"It appears that you are running on Github Actions, however Captain is unable to determine your job name.",
				"You can configure Github's job name by setting the GITHUB_JOB environment variable.",
			)
		}

		if cfg.Repository == "" {
			return errors.NewConfigurationError(
				"Missing repository",
				"It appears that you are running on Github Actions, however Captain is unable to determine your repository name.",
				"You can configure the repository name by setting the GITHUB_REPOSITORY environment variable.",
			)
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
		"github_pr_title":        eventPayloadData.PullRequest.Title,
		"github_pr_number":       eventPayloadData.PullRequest.Number,
	}

	return tags, err
}
