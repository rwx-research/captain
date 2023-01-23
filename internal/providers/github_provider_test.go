package providers_test

import (
	"encoding/json"

	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GithubProvider.Validate", func() {
	It("is valid", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "test",
			AccountName:    "rwx",
			JobName:        "some-job",
			RunAttempt:     "1",
			RunID:          "123",
			RepositoryName: "captain-cli",
			BranchName:     "main",
			CommitSha:      "abc123",
			CommitMessage:  "fixed it",
		}
		Expect(github.Validate()).To(BeNil())
		Expect(github.GetAttemptedBy()).To(Equal("test"))
		Expect(github.GetBranchName()).To(Equal("main"))
		Expect(github.GetCommitSha()).To(Equal("abc123"))
		Expect(github.GetCommitMessage()).To(Equal("fixed it"))
		Expect(github.GetProviderName()).To(Equal("github"))
	})

	It("requires an attempted by", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "",
			AccountName:    "rwx",
			JobName:        "some-job",
			RunAttempt:     "1",
			RunID:          "123",
			RepositoryName: "captain-cli",
			BranchName:     "main",
			CommitSha:      "abc123",
		}
		err := github.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing attempted by"))
	})

	It("requires an account name", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "test",
			AccountName:    "",
			JobName:        "some-job",
			RunAttempt:     "1",
			RunID:          "123",
			RepositoryName: "captain-cli",
			BranchName:     "main",
			CommitSha:      "abc123",
		}
		err := github.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing account name"))
	})

	It("requires an job name", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "test",
			AccountName:    "rwx",
			JobName:        "",
			RunAttempt:     "1",
			RunID:          "123",
			RepositoryName: "captain-cli",
			BranchName:     "main",
			CommitSha:      "abc123",
		}
		err := github.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing job name"))
	})

	It("requires a run attempt name", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "test",
			AccountName:    "rwx",
			JobName:        "some-job",
			RunAttempt:     "",
			RunID:          "123",
			RepositoryName: "captain-cli",
			BranchName:     "main",
			CommitSha:      "abc123",
		}
		err := github.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing run attempt"))
	})

	It("requires a run ID", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "test",
			AccountName:    "rwx",
			JobName:        "some-job",
			RunAttempt:     "1",
			RunID:          "",
			RepositoryName: "captain",
			BranchName:     "main",
			CommitSha:      "abc123",
		}
		err := github.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing run ID"))
	})

	It("requires a repo name", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "test",
			AccountName:    "rwx",
			JobName:        "some-job",
			RunAttempt:     "1",
			RunID:          "123",
			RepositoryName: "",
			BranchName:     "main",
			CommitSha:      "abc123",
		}
		err := github.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing repository name"))
	})

	It("requires a branch name", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "test",
			AccountName:    "rwx",
			JobName:        "some-job",
			RunAttempt:     "1",
			RunID:          "123",
			RepositoryName: "captain-cli",
			BranchName:     "",
			CommitSha:      "abc123",
		}
		err := github.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing branch name"))
	})

	It("requires a commit sha", func() {
		github := providers.GithubProvider{
			AttemptedBy:    "test",
			AccountName:    "rwx",
			JobName:        "some-job",
			RunAttempt:     "1",
			RunID:          "123",
			RepositoryName: "captain-cli",
			BranchName:     "main",
			CommitSha:      "",
		}
		err := github.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing commit sha"))
	})
})

var _ = Describe("GithubProvider.GetJobTags", func() {
	It("constructs job tags without job matrix", func() {
		provider := providers.GithubProvider{
			RunID:          "my_run_id",
			RunAttempt:     "my_run_attempt",
			RepositoryName: "my_repo_name",
			AccountName:    "my_account_name",
			JobName:        "my_job_name",
			JobMatrix:      "",
		}
		Expect(provider.GetJobTags()).To(Equal(map[string]any{
			"github_run_id":          "my_run_id",
			"github_run_attempt":     "my_run_attempt",
			"github_repository_name": "my_repo_name",
			"github_account_owner":   "my_account_name",
			"github_job_name":        "my_job_name",
		}))
	})

	It("constructs job tags with job matrix", func() {
		provider := providers.GithubProvider{
			RunID:          "my_run_id",
			RunAttempt:     "my_run_attempt",
			RepositoryName: "my_repo_name",
			AccountName:    "my_account_name",
			JobName:        "my_job_name",
			JobMatrix:      "{ \"foo\": \"bar\"}",
		}
		matrix := json.RawMessage("{ \"foo\": \"bar\"}")
		Expect(provider.GetJobTags()).To(Equal(map[string]any{
			"github_run_id":          "my_run_id",
			"github_run_attempt":     "my_run_attempt",
			"github_repository_name": "my_repo_name",
			"github_account_owner":   "my_account_name",
			"github_job_name":        "my_job_name",
			"github_job_matrix":      &matrix,
		}))
	})
})
