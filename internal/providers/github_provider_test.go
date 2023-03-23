package providers_test

import (
	"encoding/json"

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
			Matrix:     "",
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
		Expect(err.Error()).To(ContainSubstring("missing repository"))
	})

	It("requires an job name", func() {
		params.Name = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing job name"))
	})

	It("requires a run attempt name", func() {
		params.Attempt = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing run attempt"))
	})

	It("requires a run ID", func() {
		params.ID = ""
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing run ID"))
	})

	It("requires a json object be passed as the job matrix", func() {
		params.Matrix = "0"
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			`invalid github-job-matrix value supplied.
Please provide '${{ toJSON(matrix) }}' with single quotes.
This information is required due to a limitation with GitHub.`,
		))
	})

	It("accepts a json object job matrix", func() {
		params.Matrix = `{"index": 0, "total": 4}`
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(BeNil())
	})

	It("allows a null job matrix if someone blindly copies the docs but doesnt actually have a matrix", func() {
		params.Matrix = "null"
		_, err := params.MakeProviderWithoutCommitMessageParsing("fixed it")
		Expect(err).To(BeNil())
	})
})

var _ = Describe("GithubProvider.JobTags", func() {
	It("constructs job tags without job matrix", func() {
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
			Matrix:     "",
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

	It("constructs job tags with job matrix", func() {
		provider, _ := providers.GitHubEnv{
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
			Matrix:     "{ \"foo\": \"bar\"}",
		}.MakeProviderWithoutCommitMessageParsing("fixed it")
		matrix := json.RawMessage("{ \"foo\": \"bar\"}")
		Expect(provider.JobTags).To(Equal(map[string]any{
			"github_run_id":          "my_run_id",
			"github_run_attempt":     "my_run_attempt",
			"github_repository_name": "my_repo_name",
			"github_account_owner":   "my_account_name",
			"github_job_name":        "my_job_name",
			"github_job_matrix":      &matrix,
		}))
	})
})
