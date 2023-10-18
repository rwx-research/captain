package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GenericEnv.MakeProvider", func() {
	It("constructs job tags", func() {
		provider := providers.GenericEnv{
			Who:           "test",
			Branch:        "testing-env-vars",
			Sha:           "abc123",
			CommitMessage: "Testing env vars -- the commit message",
			BuildURL:      "https://jenkins.example.com/job/test/123/",
			Title:         "Some title",
		}.MakeProvider()

		Expect(provider.JobTags).To(Equal(map[string]any{"captain_build_url": "https://jenkins.example.com/job/test/123/"}))
	})
})

var _ = Describe("MergeGeneric", func() {
	It("merges Who", func() {
		Expect(providers.MergeGeneric(providers.GenericEnv{Who: "p1"},
			providers.GenericEnv{Who: "p2"}).Who).To(Equal("p2"))
		Expect(providers.MergeGeneric(providers.GenericEnv{Who: "p1"},
			providers.GenericEnv{}).Who).To(Equal("p1"))
	})

	It("merges Branch", func() {
		Expect(providers.MergeGeneric(providers.GenericEnv{Branch: "p1"},
			providers.GenericEnv{Branch: "p2"}).Branch).To(Equal("p2"))
		Expect(providers.MergeGeneric(providers.GenericEnv{Branch: "p1"},
			providers.GenericEnv{}).Branch).To(Equal("p1"))
	})

	It("merges Sha", func() {
		Expect(providers.MergeGeneric(providers.GenericEnv{Sha: "p1"},
			providers.GenericEnv{Sha: "p2"}).Sha).To(Equal("p2"))
		Expect(providers.MergeGeneric(providers.GenericEnv{Sha: "p1"},
			providers.GenericEnv{}).Sha).To(Equal("p1"))
	})

	It("merges CommitMessage", func() {
		Expect(providers.MergeGeneric(providers.GenericEnv{CommitMessage: "p1"},
			providers.GenericEnv{CommitMessage: "p2"}).CommitMessage).To(Equal("p2"))
		Expect(providers.MergeGeneric(providers.GenericEnv{CommitMessage: "p1"},
			providers.GenericEnv{}).CommitMessage).To(Equal("p1"))
	})

	It("merges BuildURL", func() {
		Expect(providers.MergeGeneric(providers.GenericEnv{BuildURL: "p1"},
			providers.GenericEnv{BuildURL: "p2"}).BuildURL).To(Equal("p2"))
		Expect(providers.MergeGeneric(providers.GenericEnv{BuildURL: "p1"},
			providers.GenericEnv{}).BuildURL).To(Equal("p1"))
	})

	It("merges Title", func() {
		Expect(providers.MergeGeneric(providers.GenericEnv{Title: "p1"},
			providers.GenericEnv{Title: "p2"}).Title).To(Equal("p2"))
		Expect(providers.MergeGeneric(providers.GenericEnv{Title: "p1"},
			providers.GenericEnv{}).Title).To(Equal("p1"))
	})
})
