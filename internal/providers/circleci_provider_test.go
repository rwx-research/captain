package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CircleciProvider.Validate", func() {
	var provider providers.CircleciProvider

	BeforeEach(func() {
		provider = providers.CircleciProvider{
			BuildNum:         "18",
			BuildURL:         "https://circleci.com/gh/rwx-research/captain-cli/18",
			JobName:          "build",
			ParallelJobIndex: "0",
			ParallelJobTotal: "2",

			RepoAccountName: "rwx",
			RepoName:        "captain-cli",
			RepoURL:         "git@github.com:rwx-research/captain-cli.git",

			BranchName:  "main",
			CommitSha:   "000bd5713d35f778fb51d2b0bf034e8fff5b60b1",
			AttemptedBy: "test",
		}
	})

	It("is valid", func() {
		Expect(provider.Validate()).To(BeNil())
		Expect(provider.GetAttemptedBy()).To(Equal("test"))
		Expect(provider.GetBranchName()).To(Equal("main"))
		Expect(provider.GetCommitSha()).To(Equal("000bd5713d35f778fb51d2b0bf034e8fff5b60b1"))
		Expect(provider.GetCommitMessage()).To(Equal(""))
		Expect(provider.GetProviderName()).To(Equal("circleci"))
	})

	It("requires BuildNum", func() {
		provider.BuildNum = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildNum"))
	})

	It("requires BuildURL", func() {
		provider.BuildURL = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildURL"))
	})

	It("requires JobName", func() {
		provider.JobName = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobName"))
	})

	It("does not require ParallelJobIndex", func() {
		provider.ParallelJobIndex = ""
		err := provider.Validate()
		Expect(err).ToNot(HaveOccurred())
	})

	It("does not require ParallelJobTotal", func() {
		provider.ParallelJobTotal = ""
		err := provider.Validate()
		Expect(err).ToNot(HaveOccurred())
	})

	It("requires RepoAccountName", func() {
		provider.RepoAccountName = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing RepoAccountName"))
	})

	It("requires RepoName", func() {
		provider.RepoName = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing RepoName"))
	})

	It("requires RepoURL", func() {
		provider.RepoURL = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing RepoURL"))
	})

	It("requires BranchName", func() {
		provider.BranchName = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BranchName"))
	})

	It("requires CommitSha", func() {
		provider.CommitSha = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing CommitSha"))
	})

	It("requires AttemptedBy", func() {
		provider.AttemptedBy = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing AttemptedBy"))
	})
})

var _ = Describe("CircleciProvider.GetJobTags", func() {
	It("constructs job tags with parallel attributes", func() {
		provider := providers.CircleciProvider{
			BuildNum:         "18",
			BuildURL:         "https://circleci.com/gh/rwx-research/captain-cli/18",
			JobName:          "build",
			ParallelJobIndex: "0",
			ParallelJobTotal: "2",

			RepoAccountName: "rwx",
			RepoName:        "captain-cli",
			RepoURL:         "git@github.com:rwx-research/captain-cli.git",

			BranchName:  "main",
			CommitSha:   "000bd5713d35f778fb51d2b0bf034e8fff5b60b1",
			AttemptedBy: "test",
		}

		Expect(provider.GetJobTags()).To(Equal(map[string]any{
			"circle_build_num":        "18",
			"circle_build_url":        "https://circleci.com/gh/rwx-research/captain-cli/18",
			"circle_job":              "build",
			"circle_repository_url":   "git@github.com:rwx-research/captain-cli.git",
			"circle_project_username": "rwx",
			"circle_project_reponame": "captain-cli",
			"circle_node_index":       "0",
			"circle_node_total":       "2",
		}))
	})

	It("constructs job tags without parallel attributes", func() {
		provider := providers.CircleciProvider{
			BuildNum:         "18",
			BuildURL:         "https://circleci.com/gh/rwx-research/captain-cli/18",
			JobName:          "build",
			ParallelJobIndex: "",
			ParallelJobTotal: "",

			RepoAccountName: "rwx",
			RepoName:        "captain-cli",
			RepoURL:         "git@github.com:rwx-research/captain-cli.git",

			BranchName:  "main",
			CommitSha:   "000bd5713d35f778fb51d2b0bf034e8fff5b60b1",
			AttemptedBy: "test",
		}

		Expect(provider.GetJobTags()).To(Equal(map[string]any{
			"circle_build_num":        "18",
			"circle_build_url":        "https://circleci.com/gh/rwx-research/captain-cli/18",
			"circle_job":              "build",
			"circle_repository_url":   "git@github.com:rwx-research/captain-cli.git",
			"circle_project_username": "rwx",
			"circle_project_reponame": "captain-cli",
		}))
	})
})
