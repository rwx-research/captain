package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/config"
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
			Title:        "some-title",
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
			Expect(err.Error()).To(ContainSubstring("Unsupported runtime environment"))
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
			Expect(err.Error()).To(ContainSubstring("Missing 'who'"))
		})

		It("is invalid if BranchName is empty", func() {
			provider.BranchName = ""
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing branch name"))
		})
		It("is invalid if CommitSha is empty", func() {
			provider.CommitSha = ""
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing commit SHA"))
		})
		It("is VALID if CommitMessage is empty", func() {
			provider.CommitMessage = ""
			err := providers.Validate(provider)
			Expect(err).To(BeNil())
		})
		It("is VALID if Title is empty", func() {
			provider.Title = ""
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
		Title:         "some-title",
	}

	provider2 := providers.Provider{
		AttemptedBy:   "you",
		BranchName:    "test-branch",
		CommitSha:     "qrs789",
		CommitMessage: "another commit message",
		JobTags:       map[string]any{},
		ProviderName:  "GitHub",
		Title:         "another-title",
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
		Expect(merged.Title).To(Equal(provider2.Title))
	})
})

var _ = Describe("MakeProvider", func() {
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

				provider, err := env.MakeProvider()
				Expect(err).NotTo(HaveOccurred())
				Expect(provider).To(Equal(
					providers.Provider{
						ProviderName:  "generic",
						CommitSha:     "abc123",
						BranchName:    "main",
						AttemptedBy:   "me",
						CommitMessage: "",
						Title:         "",
						JobTags:       map[string]any{"captain_build_url": ""},
					},
				))
			})
		})

		Context("but with an invalid generic provider", func() {
			It("returns the invalid generic provider", func() {
				// different commands have different concepts of what a valid generic provider is
				// so we can't validate that in MakeProvider
				env.Generic.Who = "me"
				provider, err := env.MakeProvider()
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.AttemptedBy).To(Equal("me"))
			})
		})

		It("when title is set, uses the set title", func() {
			env.Generic.Title = "foo"
			provider, err := env.MakeProvider()
			Expect(err).ToNot(HaveOccurred())
			Expect(provider.Title).To(Equal("foo"))
			Expect(provider.CommitMessage).To(Equal(""))
		})

		Context("When commit message has a single line", func() {
			BeforeEach(func() {
				env.Generic.CommitMessage = "fixed it"
			})

			It("when title is unset, sets title to the commit message", func() {
				provider, err := env.MakeProvider()
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.Title).To(Equal("fixed it"))
				Expect(provider.CommitMessage).To(Equal("fixed it"))
			})
			It("when title is set, uses the set title", func() {
				env.Generic.Title = "didn't fix it yet"
				provider, err := env.MakeProvider()
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.Title).To(Equal("didn't fix it yet"))
				Expect(provider.CommitMessage).To(Equal("fixed it"))
			})
		})

		Context("When commit message has a multiple lines line", func() {
			BeforeEach(func() {
				env.Generic.CommitMessage = "fixed it\nit was tricky!"
			})

			It("when title is unset, sets title to the first line of the commit message", func() {
				provider, err := env.MakeProvider()
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.Title).To(Equal("fixed it"))
				Expect(provider.CommitMessage).To(Equal("fixed it\nit was tricky!"))
			})
			It("when title is set, uses the set title", func() {
				env.Generic.Title = "didn't fix it yet"
				provider, err := env.MakeProvider()
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.Title).To(Equal("didn't fix it yet"))
				Expect(provider.CommitMessage).To(Equal("fixed it\nit was tricky!"))
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
			provider, err := env.MakeProvider()

			Expect(err).NotTo(HaveOccurred())
			Expect(provider).To(Equal(
				providers.Provider{
					ProviderName:   "buildkite",
					CommitSha:      "abc123",
					BranchName:     "main",
					AttemptedBy:    "foo@bar.com",
					CommitMessage:  "fixed it",
					Title:          "fixed it",
					PartitionNodes: config.PartitionNodes{Index: 0, Total: 2},
					JobTags: map[string]any{
						"buildkite_retry_count":        "0",
						"buildkite_parallel_job_count": "2",
						"buildkite_build_id":           "1234",
						"buildkite_build_url":          "https://buildkit.com/builds/42",
						"buildkite_job_id":             "build123",
						"buildkite_label":              "lint",
						"buildkite_repo":               "git@github.com/rwx-research/captain-cli",
						"buildkite_organization_slug":  "rwx",
						"buildkite_parallel_job":       "0",
					},
				},
			))
		})

		Context("and with a generic provider", func() {
			It("returns the merged provider with the title from the generic provider", func() {
				env.Generic.Sha = "qrs789"
				env.Generic.CommitMessage = "fixed it on Tuesday\nthis commit message annotated before writing to captain"
				provider, err := env.MakeProvider()
				Expect(err).NotTo(HaveOccurred())
				Expect(provider).To(Equal(
					providers.Provider{
						ProviderName:   "buildkite",
						CommitSha:      "qrs789",
						BranchName:     "main",
						AttemptedBy:    "foo@bar.com",
						CommitMessage:  "fixed it on Tuesday\nthis commit message annotated before writing to captain",
						Title:          "fixed it on Tuesday",
						PartitionNodes: config.PartitionNodes{Total: 2, Index: 0},
						JobTags: map[string]any{
							"buildkite_retry_count":        "0",
							"buildkite_parallel_job_count": "2",
							"buildkite_build_id":           "1234",
							"buildkite_build_url":          "https://buildkit.com/builds/42",
							"buildkite_job_id":             "build123",
							"buildkite_label":              "lint",
							"buildkite_repo":               "git@github.com/rwx-research/captain-cli",
							"buildkite_organization_slug":  "rwx",
							"buildkite_parallel_job":       "0",
						},
					},
				))
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
			provider, _ := env.MakeProvider()
			err := providers.Validate(provider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing branch name"))
		})

		Context("and with a generic provider that will be valid after merge", func() {
			It("returns the merged provider", func() {
				env.Generic.Branch = "main"
				provider, _ := env.MakeProvider()
				err := providers.Validate(provider)
				Expect(err).ToNot(HaveOccurred())
				Expect(provider.BranchName).To(Equal("main"))
			})
		})
	})
})
