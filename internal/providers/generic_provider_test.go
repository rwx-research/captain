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
