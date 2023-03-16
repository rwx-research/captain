package main_test

import (
	captain "github.com/rwx-research/captain-cli/cmd/captain"
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MakeProviderAdapter", func() {
	var cfg *captain.Config

	BeforeEach(func() {
		cfg = &captain.Config{}
	})

	makeProvider := func() (providers.Provider, error) {
		return captain.MakeProviderAdapter(*cfg)
	}

	Context("with no detected provider info", func() {
		It("returns an error", func() {
			_, err := makeProvider()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Could not detect a supported CI provider. Without a supported CI " +
				"provider, we require --who, --branch, and --sha to be set."))
		})
		Context("but with a valid generic provider", func() {
			It("returns the generic provider", func() {
				cfg.Generic.Who = "me"
				cfg.Generic.Branch = "main"
				cfg.Generic.Sha = "abc123"

				provider, err := makeProvider()
				Expect(err).NotTo(HaveOccurred())
				Expect(provider.ProviderName).To(Equal("generic"))
				Expect(provider.CommitSha).To(Equal("abc123"))
				Expect(provider.BranchName).To(Equal("main"))
				Expect(provider.AttemptedBy).To(Equal("me"))
			})
		})

		Context("but with an invalid generic provider", func() {
			It("returns an error", func() {
				cfg.Generic.Who = "me"
				_, err := makeProvider()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Could not detect a supported CI provider. Without a supported CI " +
					"provider, we require --who, --branch, and --sha to be set."))
			})
		})
	})

	Context("with valid detected provider info", func() {
		BeforeEach(func() {
			cfg.Buildkite = providers.BuildkiteEnv{
				Detected:          true,
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
		It("returns provider info", func() {
			provider, err := makeProvider()
			Expect(err).NotTo(HaveOccurred())
			Expect(provider.ProviderName).To(Equal("buildkite"))
			Expect(provider.CommitSha).To(Equal("abc123"))
			Expect(provider.BranchName).To(Equal("main"))
			Expect(provider.AttemptedBy).To(Equal("foo@bar.com"))
		})

		Context("and with a generic provider", func() {
			It("returns the merged provider", func() {
				cfg.Generic.Sha = "qrs789"
				provider, err := makeProvider()
				Expect(err).NotTo(HaveOccurred())
				Expect(provider.ProviderName).To(Equal("buildkite"))
				Expect(provider.CommitSha).To(Equal("qrs789"))
				Expect(provider.BranchName).To(Equal("main"))
				Expect(provider.AttemptedBy).To(Equal("foo@bar.com"))
			})
		})
	})

	Context("with invalid detected provider info", func() {
		BeforeEach(func() {
			cfg.Buildkite = providers.BuildkiteEnv{
				Detected:          true,
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
		It("returns an error", func() {
			_, err := makeProvider()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing branch name"))
		})

		Context("and with a generic provider that will be valid after merge", func() {
			It("returns the merged provider", func() {
				cfg.Generic.Branch = "main"
				provider, err := makeProvider()
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.BranchName).To(Equal("main"))
			})
		})
	})
})
