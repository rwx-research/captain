package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CircleCiEnv.MakeProvider", func() {
	var params providers.CircleCIEnv

	BeforeEach(func() {
		params = providers.CircleCIEnv{
			BuildNum: "18",
			BuildURL: "https://circleci.com/gh/rwx-research/captain-cli/18",
			Job:      "build",

			NodeIndex: "0",
			NodeTotal: "2",

			ProjectUsername: "rwx",
			ProjectReponame: "captain-cli",
			RepositoryURL:   "git@github.com:rwx-research/captain-cli.git",

			Branch:   "main",
			Sha1:     "000bd5713d35f778fb51d2b0bf034e8fff5b60b1",
			Username: "test",
		}
	})

	It("is valid", func() {
		provider, err := params.MakeProvider()
		Expect(err).To(BeNil())
		Expect(provider.AttemptedBy).To(Equal("test"))
		Expect(provider.BranchName).To(Equal("main"))
		Expect(provider.CommitSha).To(Equal("000bd5713d35f778fb51d2b0bf034e8fff5b60b1"))
		Expect(provider.CommitMessage).To(Equal(""))
		Expect(provider.ProviderName).To(Equal("circleci"))
		Expect(provider.Title).To(Equal("build (18)"))
	})

	It("requires BuildNum", func() {
		params.BuildNum = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing build number"))
	})

	It("requires BuildURL", func() {
		params.BuildURL = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing build URL"))
	})

	It("requires JobName", func() {
		params.Job = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing job name"))
	})

	It("does not require NodeIndex", func() {
		params.NodeIndex = ""
		_, err := params.MakeProvider()
		Expect(err).ToNot(HaveOccurred())
	})

	It("does not require NodeTotal", func() {
		params.NodeTotal = ""
		_, err := params.MakeProvider()
		Expect(err).ToNot(HaveOccurred())
	})

	It("requires RepoAccountName", func() {
		params.ProjectUsername = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing project username"))
	})

	It("requires RepoName", func() {
		params.ProjectReponame = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing repository name"))
	})

	It("requires RepoURL", func() {
		params.RepositoryURL = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing repository URL"))
	})
})

var _ = Describe("Circleciparams.JobTags", func() {
	It("constructs job tags with parallel attributes", func() {
		provider, _ := providers.CircleCIEnv{
			BuildNum:  "18",
			BuildURL:  "https://circleci.com/gh/rwx-research/captain-cli/18",
			Job:       "build",
			NodeIndex: "0",
			NodeTotal: "2",

			ProjectUsername: "rwx",
			ProjectReponame: "captain-cli",
			RepositoryURL:   "git@github.com:rwx-research/captain-cli.git",

			Branch:   "main",
			Sha1:     "000bd5713d35f778fb51d2b0bf034e8fff5b60b1",
			Username: "test",
		}.MakeProvider()

		Expect(provider.JobTags).To(Equal(map[string]any{
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
		provider, _ := providers.CircleCIEnv{
			BuildNum:  "18",
			BuildURL:  "https://circleci.com/gh/rwx-research/captain-cli/18",
			Job:       "build",
			NodeIndex: "",
			NodeTotal: "",

			ProjectUsername: "rwx",
			ProjectReponame: "captain-cli",
			RepositoryURL:   "git@github.com:rwx-research/captain-cli.git",

			Branch:   "main",
			Sha1:     "000bd5713d35f778fb51d2b0bf034e8fff5b60b1",
			Username: "test",
		}.MakeProvider()

		Expect(provider.JobTags).To(Equal(map[string]any{
			"circle_build_num":        "18",
			"circle_build_url":        "https://circleci.com/gh/rwx-research/captain-cli/18",
			"circle_job":              "build",
			"circle_repository_url":   "git@github.com:rwx-research/captain-cli.git",
			"circle_project_username": "rwx",
			"circle_project_reponame": "captain-cli",
		}))
	})
})
