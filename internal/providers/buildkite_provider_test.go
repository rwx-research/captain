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
			Branch:            "main",
			BuildCreatorEmail: "foo@bar.com",
			BuildID:           "1234",
			RetryCount:        "0",
			BuildURL:          "https://buildkit.com/builds/42",
			Message:           "fixed it",
			Commit:            "abc123",
			JobID:             "build123",
			Label:             "lint",
			ParallelJob:       "0",
			ParallelJobCount:  "2",
			OrganizationSlug:  "rwx",
			Repo:              "git@github.com/rwx-research/captain-cli",
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
		provider.Branch = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing Branch"))
	})

	It("requires build creator email", func() {
		provider.BuildCreatorEmail = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildCreatorEmail"))
	})

	It("requires build id", func() {
		provider.BuildID = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildID"))
	})

	It("requires build retry count", func() {
		provider.RetryCount = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing RetryCount"))
	})

	It("requires build url", func() {
		provider.BuildURL = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing BuildURL"))
	})

	It("requires commit sha", func() {
		provider.Commit = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing Commit"))
	})

	It("requires job id", func() {
		provider.JobID = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing JobID"))
	})

	It("requires job name", func() {
		provider.Label = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing Label"))
	})

	It("requires buildkite org slug", func() {
		provider.OrganizationSlug = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing OrganizationSlug"))
	})

	It("requires repository url", func() {
		provider.Repo = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing Repo"))
	})
})

var _ = Describe("BuildkiteProvider.GetJobTags", func() {
	It("constructs job tags with parallel attributes", func() {
		provider := providers.BuildkiteProvider{
			Branch:            "main",
			BuildCreatorEmail: "foo@bar.com",
			BuildID:           "1234",
			RetryCount:        "0",
			BuildURL:          "https://buildkit.com/builds/42",
			Message:           "fixed it",
			Commit:            "abc123",
			JobID:             "build123",
			Label:             "lint",
			ParallelJob:       "0",
			ParallelJobCount:  "2",
			OrganizationSlug:  "rwx",
			Repo:              "git@github.com/rwx-research/captain-cli",
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
			Branch:            "main",
			BuildCreatorEmail: "foo@bar.com",
			BuildID:           "1234",
			RetryCount:        "0",
			BuildURL:          "https://buildkit.com/builds/42",
			Message:           "fixed it",
			Commit:            "abc123",
			JobID:             "build123",
			Label:             "lint",
			ParallelJob:       "",
			ParallelJobCount:  "",
			OrganizationSlug:  "rwx",
			Repo:              "git@github.com/rwx-research/captain-cli",
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
