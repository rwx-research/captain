package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GitHubEnv.MakeProvider", func() {
	var params providers.GitHubEnv
	var eventPayloadData providers.EventPayloadData

	BeforeEach(func() {
		params = providers.GitHubEnv{
			// for branch name
			EventName: "push",
			RefName:   "main",
			HeadRef:   "main",
			// for commit message
			EventPath: "/tmp/event.json",
			// attempted by
			ExecutingActor:  "test",
			TriggeringActor: "test",
			// for commit sha
			CommitSha: "abc123",
			// for tags
			ID:         "123",
			Attempt:    "1",
			Repository: "rwx/captain-cli",
			Name:       "some-job",
		}
		eventPayloadData = providers.EventPayloadData{}
		eventPayloadData.HeadCommit.Message = "fixed it\nyeah"
		eventPayloadData.PullRequest.Number = 5
		eventPayloadData.PullRequest.Title = "PR title"
	})

	It("is valid", func() {
		provider, err := params.MakeProviderWithoutCommitMessageParsing(eventPayloadData)
		Expect(err).To(BeNil())
		Expect(provider.AttemptedBy).To(Equal("test"))
		Expect(provider.BranchName).To(Equal("main"))
		Expect(provider.CommitSha).To(Equal("abc123"))
		Expect(provider.CommitMessage).To(Equal("fixed it\nyeah"))
		Expect(provider.ProviderName).To(Equal("github"))
		Expect(provider.Title).To(Equal("fixed it"))
	})

	It("uses the PR info for the title when the commit message is missing", func() {
		eventPayloadData.HeadCommit.Message = ""
		provider, err := params.MakeProviderWithoutCommitMessageParsing(eventPayloadData)
		Expect(err).To(BeNil())
		Expect(provider.AttemptedBy).To(Equal("test"))
		Expect(provider.BranchName).To(Equal("main"))
		Expect(provider.CommitSha).To(Equal("abc123"))
		Expect(provider.CommitMessage).To(Equal(""))
		Expect(provider.ProviderName).To(Equal("github"))
		Expect(provider.Title).To(Equal("PR title (PR #5)"))
	})

	It("requires an repository", func() {
		params.Repository = ""

		_, err := params.MakeProviderWithoutCommitMessageParsing(eventPayloadData)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing repository"))
	})

	It("requires an job name", func() {
		params.Name = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing(eventPayloadData)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing job name"))
	})

	It("requires a run attempt name", func() {
		params.Attempt = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing(eventPayloadData)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing run attempt"))
	})

	It("requires a run ID", func() {
		params.ID = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing(eventPayloadData)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing run ID"))
	})
})

var _ = Describe("GithubProvider.JobTags", func() {
	It("constructs job tags", func() {
		eventPayloadData := providers.EventPayloadData{}
		eventPayloadData.HeadCommit.Message = "fixed it\nyeah"
		eventPayloadData.PullRequest.Number = 5
		eventPayloadData.PullRequest.Title = "PR title"

		provider, err := providers.GitHubEnv{
			// for branch name
			EventName: "push",
			RefName:   "main",
			HeadRef:   "main",
			// for commit message
			EventPath: "/tmp/event.json",
			// attempted by
			ExecutingActor:  "test",
			TriggeringActor: "test",
			// for commit sha
			CommitSha: "abc123",

			// for tags
			ID:         "my_run_id",
			Attempt:    "my_run_attempt",
			Repository: "my_account_name/my_repo_name",
			Name:       "my_job_name",
		}.MakeProviderWithoutCommitMessageParsing(eventPayloadData)

		Expect(err).To(BeNil())

		Expect(provider.JobTags).To(Equal(map[string]any{
			"github_run_id":          "my_run_id",
			"github_run_attempt":     "my_run_attempt",
			"github_repository_name": "my_repo_name",
			"github_account_owner":   "my_account_name",
			"github_job_name":        "my_job_name",
			"github_pr_number":       5,
			"github_pr_title":        "PR title",
		}))
	})
})
