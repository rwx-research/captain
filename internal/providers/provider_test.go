package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validate", func() {
	var provider providers.Provider
	BeforeEach(func() {
		provider = providers.Provider{
			AttemptedBy:  "me",
			BranchName:   "main",
			CommitSha:    "abc123",
			JobTags:      map[string]any{},
			ProviderName: "",
		}
	})

	Context("with a generic provider", func() {
		BeforeEach(func() {
			provider.ProviderName = "generic"
		})

		It("has a special error message if invalid", func() {
			provider.AttemptedBy = ""
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Could not detect a supported CI provider. Without a " +
				"supported CI provider, we require --who, --branch, and --sha to be set."))
		})
	})

	Context("with a non-generic provider", func() {
		BeforeEach(func() {
			provider.ProviderName = "github"
		})

		It("is valid", func() {
			Expect(providers.Validate(provider)).To(BeNil())
		})

		It("is invalid if ProviderName is empty", func() {
			provider.ProviderName = ""
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("provider name should never be empty"))
		})

		It("is invalid if AttemptedBy is empty", func() {
			provider.AttemptedBy = ""
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing attempted by"))
		})

		It("is invalid if BranchName is empty", func() {
			provider.BranchName = ""
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing branch name"))
		})
		It("is invalid if CommitSha is empty", func() {
			provider.CommitSha = ""
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing commit sha"))
		})
		It("is VALID if CommitMessage is empty", func() {
			provider.CommitMessage = ""
			err := providers.Validate(provider)
			Expect(err).To(BeNil())
		})
	})
})

var _ = Describe("providers.Merge", func() {
	provider1 := providers.Provider{
		AttemptedBy:   "me",
		BranchName:    "main",
		CommitSha:     "abc123",
		CommitMessage: "one commit message",
		JobTags:       map[string]any{},
		ProviderName:  "generic",
	}

	provider2 := providers.Provider{
		AttemptedBy:   "you",
		BranchName:    "test-branch",
		CommitSha:     "qrs789",
		CommitMessage: "another commit message",
		JobTags:       map[string]any{},
		ProviderName:  "GitHub",
	}

	It("returns the second provider if the first is empty", func() {
		merged := providers.Merge(providers.Provider{}, provider1)
		Expect(merged).To(Equal(provider1))
	})

	It("changes nothing if the second provider is empty", func() {
		merged := providers.Merge(provider1, providers.Provider{})
		Expect(merged).To(Equal(provider1))
	})

	It("merges all fields except provider name", func() {
		merged := providers.Merge(provider1, provider2)
		Expect(merged.ProviderName).To(Equal(provider1.ProviderName))
		Expect(merged.AttemptedBy).To(Equal(provider2.AttemptedBy))
		Expect(merged.BranchName).To(Equal(provider2.BranchName))
		Expect(merged.CommitSha).To(Equal(provider2.CommitSha))
		Expect(merged.CommitMessage).To(Equal(provider2.CommitMessage))
	})
})

var _ = Describe("MakeProviderAdapter", func() {
	var env providers.Env

	BeforeEach(func() {
		env = providers.Env{}
	})

	Context("with no detected provider info", func() {
		Context("but with a valid generic provider", func() {
			It("returns the generic provider", func() {
				env.Generic.Who = "me"
				env.Generic.Branch = "main"
				env.Generic.Sha = "abc123"

				provider, err := env.MakeProviderAdapter()
				Expect(err).NotTo(HaveOccurred())
				Expect(provider.ProviderName).To(Equal("generic"))
				Expect(provider.CommitSha).To(Equal("abc123"))
				Expect(provider.BranchName).To(Equal("main"))
				Expect(provider.AttemptedBy).To(Equal("me"))
			})
		})

		Context("but with an invalid generic provider", func() {
			It("returns the invalid generic provider", func() {
				// different commands have different concepts of what a valid generic provider is
				// so we can't validate that in MakeProviderAdapter
				env.Generic.Who = "me"
				_, err := env.MakeProviderAdapter()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("with valid detected provider info", func() {
		BeforeEach(func() {
			env.Buildkite = providers.BuildkiteEnv{
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
			provider, err := env.MakeProviderAdapter()
			Expect(err).NotTo(HaveOccurred())
			Expect(provider.ProviderName).To(Equal("buildkite"))
			Expect(provider.CommitSha).To(Equal("abc123"))
			Expect(provider.BranchName).To(Equal("main"))
			Expect(provider.AttemptedBy).To(Equal("foo@bar.com"))
		})

		Context("and with a generic provider", func() {
			It("returns the merged provider", func() {
				env.Generic.Sha = "qrs789"
				provider, err := env.MakeProviderAdapter()
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
			env.Buildkite = providers.BuildkiteEnv{
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
		It("is invalid", func() {
			provider, _ := env.MakeProviderAdapter()
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing branch name"))
		})

		Context("and with a generic provider that will be valid after merge", func() {
			It("returns the merged provider", func() {
				env.Generic.Branch = "main"
				provider, _ := env.MakeProviderAdapter()
				err := providers.Validate(provider)
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.BranchName).To(Equal("main"))
			})
		})
	})
})
