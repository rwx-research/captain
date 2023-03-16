package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("providers.Validate", func() {
	var provider providers.Provider
	BeforeEach(func() {
		provider = providers.Provider{
			AttemptedBy:  "me",
			BranchName:   "main",
			CommitSha:    "abc123",
			JobTags:      map[string]any{},
			ProviderName: "generic",
		}
	})

	It("is valid", func() {
		Expect(provider.Validate()).To(BeNil())
	})

	It("is invalid if ProviderName is empty", func() {
		provider.ProviderName = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("provider name should never be empty"))
	})

	It("is invalid if AttemptedBy is empty", func() {
		provider.AttemptedBy = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing attempted by"))
	})
	It("is invalid if BranchName is empty", func() {
		provider.BranchName = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing branch name"))
	})
	It("is invalid if CommitSha is empty", func() {
		provider.CommitSha = ""
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing commit sha"))
	})
	It("is VALID if CommitMessage is empty", func() {
		provider.CommitMessage = ""
		err := provider.Validate()
		Expect(err).To(BeNil())
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
