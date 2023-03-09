package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GitLabCIProvider.Validate", func() {
	var provider providers.GitLabCIProvider

	BeforeEach(func() {
		provider = providers.GitLabCIProvider{
			JobName:       "rspec 1/2",
			JobStage:      "test",
			JobID:         "3889399980",
			PipelineID:    "798778026",
			JobURL:        "https://gitlab.com/captain-examples/rspec/-/jobs/3889399980",
			PipelineURL:   "https://gitlab.com/captain-examples/rspec/-/pipelines/798778026",
			AttemptedBy:   "michaelglass",
			NodeTotal:     "2",
			NodeIndex:     "1",
			RepoPath:      "captain-examples/rspec",
			ProjectURL:    "https://gitlab.com/captain-examples/rspec",
			CommitSHA:     "03d68a49ef1e131cf8942d5a07a0ff008ede6a1a",
			CommitAuthor:  "Michael Glass <me@mike.is>",
			BranchName:    "main",
			CommitMessage: "this is what I did\nand here are some details",
			APIURL:        "https://gitlab.com/api/v4",
		}
	})

	It("is valid", func() {
		Expect(provider.Validate()).To(BeNil())
		Expect(provider.GetAttemptedBy()).To(Equal("michaelglass"))
		Expect(provider.GetBranchName()).To(Equal("main"))
		Expect(provider.GetCommitSha()).To(Equal("03d68a49ef1e131cf8942d5a07a0ff008ede6a1a"))
		Expect(provider.GetCommitMessage()).To(Equal("this is what I did\nand here are some details"))
		Expect(provider.GetProviderName()).To(Equal("gitlabci"))
	})

	It("falls back to commit author if attempted by is not set", func() {
		provider.AttemptedBy = ""
		Expect(provider.GetAttemptedBy()).To(Equal("Michael Glass <me@mike.is>"))
	})

	It("requires JobName", func() {
		provider.JobName = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobName"))
	})

	It("requires JobStage", func() {
		provider.JobStage = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobStage"))
	})

	It("requires JobID", func() {
		provider.JobID = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobID"))
	})

	It("requires PipelineID", func() {
		provider.PipelineID = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing PipelineID"))
	})

	It("requires JobURL", func() {
		provider.JobURL = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobURL"))
	})

	It("requires PipelineURL", func() {
		provider.PipelineURL = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing PipelineURL"))
	})

	It("doesn't requires AttemptedBy", func() {
		provider.AttemptedBy = ""
		err := provider.Validate()

		Expect(err).ToNot(HaveOccurred())
	})

	It("doesn't requires NodeIndex", func() {
		provider.NodeIndex = ""
		err := provider.Validate()
		Expect(err).ToNot(HaveOccurred())
	})

	It("requires NodeTotal", func() {
		provider.NodeTotal = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing NodeTotal"))
	})

	It("requires RepoPath", func() {
		provider.RepoPath = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing RepoPath"))
	})

	It("requires ProjectURL", func() {
		provider.ProjectURL = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing ProjectURL"))
	})

	It("requires CommitSHA", func() {
		provider.CommitSHA = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing CommitSHA"))
	})

	It("requires CommitAuthor", func() {
		provider.CommitAuthor = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing CommitAuthor"))
	})

	It("requires BranchName", func() {
		provider.BranchName = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BranchName"))
	})

	It("requires CommitMessage", func() {
		provider.CommitMessage = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing CommitMessage"))
	})

	It("requires APIURL", func() {
		provider.APIURL = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing APIURL"))
	})
})

var _ = Describe("GitLabCIProvider.GetJobTags", func() {
	It("constructs job tags with parallel attributes", func() {
		provider := providers.GitLabCIProvider{
			JobName:       "rspec 1/2",
			JobStage:      "test",
			JobID:         "3889399980",
			PipelineID:    "798778026",
			JobURL:        "https://gitlab.com/captain-examples/rspec/-/jobs/3889399980",
			PipelineURL:   "https://gitlab.com/captain-examples/rspec/-/pipelines/798778026",
			AttemptedBy:   "michaelglass",
			NodeTotal:     "2",
			NodeIndex:     "1",
			RepoPath:      "captain-examples/rspec",
			ProjectURL:    "https://gitlab.com/captain-examples/rspec",
			CommitSHA:     "03d68a49ef1e131cf8942d5a07a0ff008ede6a1a",
			CommitAuthor:  "Michael Glass <me@mike.is>",
			BranchName:    "main",
			CommitMessage: "this is what I did\nand here are some details",
			APIURL:        "https://gitlab.com/api/v4",
		}

		Expect(provider.GetJobTags()).To(Equal(map[string]any{
			"gitlab_project_url":     "https://gitlab.com/captain-examples/rspec",
			"gitlab_job_name":        "rspec 1/2",
			"gitlab_job_stage":       "test",
			"gitlab_job_id":          "3889399980",
			"gitlab_pipeline_id":     "798778026",
			"gitlab_job_url":         "https://gitlab.com/captain-examples/rspec/-/jobs/3889399980",
			"gitlab_pipeline_url":    "https://gitlab.com/captain-examples/rspec/-/pipelines/798778026",
			"gitlab_repository_path": "captain-examples/rspec",
			"gitlab_api_v4_url":      "https://gitlab.com/api/v4",
			"gitlab_node_index":      "1",
			"gitlab_node_total":      "2",
		}))
	})

	It("constructs job tags without parallel attributes", func() {
		provider := providers.GitLabCIProvider{
			JobName:       "rspec",
			JobStage:      "test",
			JobID:         "3889399980",
			PipelineID:    "798778026",
			JobURL:        "https://gitlab.com/captain-examples/rspec/-/jobs/3889399980",
			PipelineURL:   "https://gitlab.com/captain-examples/rspec/-/pipelines/798778026",
			AttemptedBy:   "michaelglass",
			NodeTotal:     "1",
			NodeIndex:     "",
			RepoPath:      "captain-examples/rspec",
			ProjectURL:    "https://gitlab.com/captain-examples/rspec",
			CommitSHA:     "03d68a49ef1e131cf8942d5a07a0ff008ede6a1a",
			CommitAuthor:  "Michael Glass <me@mike.is>",
			BranchName:    "main",
			CommitMessage: "this is what I did\nand here are some details",
			APIURL:        "https://gitlab.com/api/v4",
		}

		Expect(provider.GetJobTags()).To(Equal(map[string]any{
			"gitlab_job_stage":       "test",
			"gitlab_job_id":          "3889399980",
			"gitlab_pipeline_id":     "798778026",
			"gitlab_job_url":         "https://gitlab.com/captain-examples/rspec/-/jobs/3889399980",
			"gitlab_pipeline_url":    "https://gitlab.com/captain-examples/rspec/-/pipelines/798778026",
			"gitlab_repository_path": "captain-examples/rspec",
			"gitlab_project_url":     "https://gitlab.com/captain-examples/rspec",
			"gitlab_job_name":        "rspec",
			"gitlab_api_v4_url":      "https://gitlab.com/api/v4",
		}))
	})
})
