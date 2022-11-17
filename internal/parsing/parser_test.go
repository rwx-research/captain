package parsing_test

import (
	"github.com/rwx-research/captain-cli/internal/parsing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser", func() {
	Describe("ParseResultSentiment", func() {
		It("has a string representation", func() {
			Expect(parsing.PositiveParseResultSentiment.String()).To(Equal("Positive"))
			Expect(parsing.NeutralParseResultSentiment.String()).To(Equal("Neutral"))
			Expect(parsing.NegativeParseResultSentiment.String()).To(Equal("Negative"))
			Expect(parsing.ParseResultSentiment(100).String()).To(Equal("Unknown (100)"))
		})
	})
})
