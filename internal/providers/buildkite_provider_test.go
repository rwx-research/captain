package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildkiteEnv.MakeProvider", func() {
	var params providers.BuildkiteEnv

	BeforeEach(func() {
		params = providers.BuildkiteEnv{
			Branch:            "main",
			BuildCreatorEmail: "foo@bar.com",
			BuildID:           "1234",
			RetryCount:        "0",
			BuildURL:          "https://buildkit.com/builds/42",
			Message:           "fixed it\nyeah",
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
		provider, err := params.MakeProvider()
		Expect(err).To(BeNil())
		Expect(provider.AttemptedBy).To(Equal("foo@bar.com"))
		Expect(provider.BranchName).To(Equal("main"))
		Expect(provider.CommitSha).To(Equal("abc123"))
		Expect(provider.CommitMessage).To(Equal("fixed it\nyeah"))
		Expect(provider.ProviderName).To(Equal("buildkite"))
		Expect(provider.Title).To(Equal("fixed it"))
	})

	It("requires build id", func() {
		params.BuildID = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing build ID"))
	})

	It("requires build retry count", func() {
		params.RetryCount = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing retry count"))
	})

	It("requires build url", func() {
		params.BuildURL = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing build URL"))
	})

	It("requires job id", func() {
		params.JobID = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing job ID"))
	})

	It("requires job name", func() {
		params.Label = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing label"))
	})

	It("requires buildkite org slug", func() {
		params.OrganizationSlug = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing organization slug"))
	})

	It("requires repository url", func() {
		params.Repo = ""
		_, err := params.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing repository"))
	})
})

var _ = Describe("BuildkiteProvider.JobTags", func() {
	It("constructs job tags with parallel attributes", func() {
		provider, _ := providers.BuildkiteEnv{
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
		}.MakeProvider()
		Expect(provider.JobTags).To(Equal(map[string]any{
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
		provider, _ := providers.BuildkiteEnv{
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
		}.MakeProvider()
		Expect(provider.JobTags).To(Equal(map[string]any{
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
