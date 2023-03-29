package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GitHubEnv.MakeProvider", func() {
	var params providers.GitHubEnv

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
	})

	It("is valid", func() {
		provider, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(BeNil())
		Expect(provider.AttemptedBy).To(Equal("test"))
		Expect(provider.BranchName).To(Equal("main"))
		Expect(provider.CommitSha).To(Equal("abc123"))
		Expect(provider.CommitMessage).To(Equal("fixed it"))
		Expect(provider.ProviderName).To(Equal("github"))
	})

	It("requires an repository", func() {
		params.Repository = ""

		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing repository"))
	})

	It("requires an job name", func() {
		params.Name = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing job name"))
	})

	It("requires a run attempt name", func() {
		params.Attempt = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing run attempt"))
	})

	It("requires a run ID", func() {
		params.ID = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing run ID"))
	})
})

var _ = Describe("GithubProvider.JobTags", func() {
	It("constructs job tags", func() {
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
		}.MakeProviderWithoutCommitMessageParsing("fixed it")

		Expect(err).To(BeNil())

		Expect(provider.JobTags).To(Equal(map[string]any{
			"github_run_id":          "my_run_id",
			"github_run_attempt":     "my_run_attempt",
			"github_repository_name": "my_repo_name",
			"github_account_owner":   "my_account_name",
			"github_job_name":        "my_job_name",
		}))
	})
})
