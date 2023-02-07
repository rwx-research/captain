package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildkiteProvider.Validate", func() {
	var provider providers.BuildkiteProvider

	BeforeEach(func() {
		provider = providers.BuildkiteProvider{
			Env: providers.BuildkiteEnv{
				BuildkiteBranch:            "main",
				BuildkiteBuildCreatorEmail: "foo@bar.com",
				BuildkiteBuildID:           "1234",
				BuildkiteRetryCount:        "0",
				BuildkiteBuildURL:          "https://buildkit.com/builds/42",
				BuildkiteMessage:           "fixed it",
				BuildkiteCommit:            "abc123",
				BuildkiteJobID:             "build123",
				BuildkiteLabel:             "lint",
				BuildkiteParallelJob:       "0",
				BuildkiteParallelJobCount:  "2",
				BuildkiteOrganizationSlug:  "rwx",
				BuildkiteRepo:              "git@github.com/rwx-research/captain-cli",
			},
		}
	})

	It("is valid", func() {
		Expect(provider.Validate()).To(BeNil())
		Expect(provider.GetAttemptedBy()).To(Equal("foo@bar.com"))
		Expect(provider.GetBranchName()).To(Equal("main"))
		Expect(provider.GetCommitSha()).To(Equal("abc123"))
		Expect(provider.GetCommitMessage()).To(Equal("fixed it"))
		Expect(provider.GetProviderName()).To(Equal("buildkite"))
	})

	It("requires branch name", func() {
		provider.Env.BuildkiteBranch = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteBranch"))
	})

	It("requires build creator email", func() {
		provider.Env.BuildkiteBuildCreatorEmail = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteBuildCreatorEmail"))
	})

	It("requires build id", func() {
		provider.Env.BuildkiteBuildID = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteBuildID"))
	})

	It("requires build retry count", func() {
		provider.Env.BuildkiteRetryCount = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteRetryCount"))
	})

	It("requires build url", func() {
		provider.Env.BuildkiteBuildURL = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteBuildURL"))
	})

	It("requires commit sha", func() {
		provider.Env.BuildkiteCommit = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteCommit"))
	})

	It("requires job id", func() {
		provider.Env.BuildkiteJobID = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteJobID"))
	})

	It("requires job name", func() {
		provider.Env.BuildkiteLabel = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteLabel"))
	})

	It("requires buildkite org slug", func() {
		provider.Env.BuildkiteOrganizationSlug = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteOrganizationSlug"))
	})

	It("requires repository url", func() {
		provider.Env.BuildkiteRepo = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildkiteRepo"))
	})
})

var _ = Describe("BuildkiteProvider.GetJobTags", func() {
	It("constructs job tags with parallel attributes", func() {
		provider := providers.BuildkiteProvider{
			Env: providers.BuildkiteEnv{
				BuildkiteBranch:            "main",
				BuildkiteBuildCreatorEmail: "foo@bar.com",
				BuildkiteBuildID:           "1234",
				BuildkiteRetryCount:        "0",
				BuildkiteBuildURL:          "https://buildkit.com/builds/42",
				BuildkiteMessage:           "fixed it",
				BuildkiteCommit:            "abc123",
				BuildkiteJobID:             "build123",
				BuildkiteLabel:             "lint",
				BuildkiteParallelJob:       "0",
				BuildkiteParallelJobCount:  "2",
				BuildkiteOrganizationSlug:  "rwx",
				BuildkiteRepo:              "git@github.com/rwx-research/captain-cli",
			},
		}
		Expect(provider.GetJobTags()).To(Equal(map[string]any{
			"buildkite_retry_count":        "0",
			"buildkite_repo":               "git@github.com/rwx-research/captain-cli",
			"buildkite_job_id":             "build123",
			"buildkite_label":              "lint",
			"buildkite_parallel_job":       "0",
			"buildkite_parallel_job_count": "2",
			"buildkite_organization_slug":  "rwx",
			"buildkite_build_id":           "1234",
			"buildkite_build_url":          "https://buildkit.com/builds/42",
		}))
	})

	It("constructs job tags without parallel attributes", func() {
		provider := providers.BuildkiteProvider{
			Env: providers.BuildkiteEnv{
				BuildkiteBranch:            "main",
				BuildkiteBuildCreatorEmail: "foo@bar.com",
				BuildkiteBuildID:           "1234",
				BuildkiteRetryCount:        "0",
				BuildkiteBuildURL:          "https://buildkit.com/builds/42",
				BuildkiteMessage:           "fixed it",
				BuildkiteCommit:            "abc123",
				BuildkiteJobID:             "build123",
				BuildkiteLabel:             "lint",
				BuildkiteParallelJob:       "",
				BuildkiteParallelJobCount:  "",
				BuildkiteOrganizationSlug:  "rwx",
				BuildkiteRepo:              "git@github.com/rwx-research/captain-cli",
			},
		}
		Expect(provider.GetJobTags()).To(Equal(map[string]any{
			"buildkite_retry_count":       "0",
			"buildkite_repo":              "git@github.com/rwx-research/captain-cli",
			"buildkite_job_id":            "build123",
			"buildkite_label":             "lint",
			"buildkite_organization_slug": "rwx",
			"buildkite_build_id":          "1234",
			"buildkite_build_url":         "https://buildkit.com/builds/42",
		}))
	})
})
