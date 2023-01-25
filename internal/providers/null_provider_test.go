package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NullProvider.Validate", func() {
	It("is never valid", func() {
		provider := providers.NullProvider{}
		err := provider.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("We've failed to detect a supported CI context"))
	})
})
