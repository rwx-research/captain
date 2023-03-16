package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GitLabCIprovider().ValidationError", func() {
	var params providers.GitLabEnv

	BeforeEach(func() {
		params = providers.GitLabEnv{
			JobName:       "rspec 1/2",
			JobStage:      "test",
			JobID:         "3889399980",
			PipelineID:    "798778026",
			JobURL:        "https://gitlab.com/captain-examples/rspec/-/jobs/3889399980",
			PipelineURL:   "https://gitlab.com/captain-examples/rspec/-/pipelines/798778026",
			UserLogin:     "michaelglass",
			NodeTotal:     "2",
			NodeIndex:     "1",
			ProjectPath:   "captain-examples/rspec",
			ProjectURL:    "https://gitlab.com/captain-examples/rspec",
			CommitSHA:     "03d68a49ef1e131cf8942d5a07a0ff008ede6a1a",
			CommitAuthor:  "Michael Glass <me@mike.is>",
			CommitBranch:  "main",
			CommitMessage: "this is what I did\nand here are some details",
			APIV4URL:      "https://gitlab.com/api/v4",
		}
	})

	It("is valid", func() {
		provider, err := providers.MakeGitLabProvider(params)
		Expect(err).To(BeNil())
		Expect(provider.AttemptedBy).To(Equal("michaelglass"))
		Expect(provider.BranchName).To(Equal("main"))
		Expect(provider.CommitSha).To(Equal("03d68a49ef1e131cf8942d5a07a0ff008ede6a1a"))
		Expect(provider.CommitMessage).To(Equal("this is what I did\nand here are some details"))
		Expect(provider.ProviderName).To(Equal("gitlabci"))
	})

	It("falls back to commit author if attempted by is not set", func() {
		params.UserLogin = ""
		provider, _ := providers.MakeGitLabProvider(params)

		Expect(provider.AttemptedBy).To(Equal("Michael Glass <me@mike.is>"))
	})

	It("requires JobName", func() {
		params.JobName = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobName"))
	})

	It("requires JobStage", func() {
		params.JobStage = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobStage"))
	})

	It("requires JobID", func() {
		params.JobID = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobID"))
	})

	It("requires PipelineID", func() {
		params.PipelineID = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing PipelineID"))
	})

	It("requires JobURL", func() {
		params.JobURL = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobURL"))
	})

	It("requires PipelineURL", func() {
		params.PipelineURL = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing PipelineURL"))
	})

	It("doesn't requires AttemptedBy", func() {
		params.UserLogin = ""
		_, err := providers.MakeGitLabProvider(params)

		Expect(err).ToNot(HaveOccurred())
	})

	It("doesn't requires NodeIndex", func() {
		params.NodeIndex = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).ToNot(HaveOccurred())
	})

	It("requires NodeTotal", func() {
		params.NodeTotal = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing NodeTotal"))
	})

	It("requires project path", func() {
		params.ProjectPath = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing project path"))
	})

	It("requires ProjectURL", func() {
		params.ProjectURL = ""
		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing ProjectURL"))
	})

	It("requires API URL", func() {
		params.APIV4URL = ""

		_, err := providers.MakeGitLabProvider(params)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing API URL"))
	})
})

var _ = Describe("GitLabCIparams.GetJobTags", func() {
	It("constructs job tags with parallel attributes", func() {
		provider, _ := providers.MakeGitLabProvider(
			providers.GitLabEnv{
				JobName:       "rspec 1/2",
				JobStage:      "test",
				JobID:         "3889399980",
				PipelineID:    "798778026",
				JobURL:        "https://gitlab.com/captain-examples/rspec/-/jobs/3889399980",
				PipelineURL:   "https://gitlab.com/captain-examples/rspec/-/pipelines/798778026",
				UserLogin:     "michaelglass",
				NodeTotal:     "2",
				NodeIndex:     "1",
				ProjectPath:   "captain-examples/rspec",
				ProjectURL:    "https://gitlab.com/captain-examples/rspec",
				CommitSHA:     "03d68a49ef1e131cf8942d5a07a0ff008ede6a1a",
				CommitAuthor:  "Michael Glass <me@mike.is>",
				CommitBranch:  "main",
				CommitMessage: "this is what I did\nand here are some details",
				APIV4URL:      "https://gitlab.com/api/v4",
			},
		)

		Expect(provider.JobTags).To(Equal(map[string]any{
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
		provider, _ := providers.MakeGitLabProvider(
			providers.GitLabEnv{
				JobName:       "rspec",
				JobStage:      "test",
				JobID:         "3889399980",
				PipelineID:    "798778026",
				JobURL:        "https://gitlab.com/captain-examples/rspec/-/jobs/3889399980",
				PipelineURL:   "https://gitlab.com/captain-examples/rspec/-/pipelines/798778026",
				UserLogin:     "michaelglass",
				NodeTotal:     "1",
				NodeIndex:     "",
				ProjectPath:   "captain-examples/rspec",
				ProjectURL:    "https://gitlab.com/captain-examples/rspec",
				CommitSHA:     "03d68a49ef1e131cf8942d5a07a0ff008ede6a1a",
				CommitAuthor:  "Michael Glass <me@mike.is>",
				CommitBranch:  "main",
				CommitMessage: "this is what I did\nand here are some details",
				APIV4URL:      "https://gitlab.com/api/v4",
			})

		Expect(provider.JobTags).To(Equal(map[string]any{
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
