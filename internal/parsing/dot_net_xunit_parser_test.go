package parsing_test

import (
	"os"
	"strings"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DotNetxUnitParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/xunit_dot_net.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.DotNetxUnitParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			Expect(parseResult.Parser).To(Equal(parsing.DotNetxUnitParser{}))
			Expect(parseResult.Sentiment).To(Equal(parsing.PositiveParseResultSentiment))
			Expect(parseResult.TestResults.Framework.Language).To(Equal(v1.FrameworkLanguageDotNet))
			Expect(parseResult.TestResults.Framework.Kind).To(Equal(v1.FrameworkKindxUnit))
			Expect(parseResult.TestResults.Summary.Tests).To(Equal(0))
			Expect(parseResult.TestResults.Summary.Successful).To(Equal(0))
			Expect(parseResult.TestResults.Summary.Failed).To(Equal(0))
			Expect(parseResult.TestResults.Summary.OtherErrors).To(Equal(0))
		})

		It("errors on malformed XML", func() {
			parseResult, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(parseResult).To(BeNil())
		})

		It("errors on XML that doesn't look like xUnit.NET", func() {
			var parseResult *parsing.ParseResult
			var err error

			parseResult, err = parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<assemblies></assemblies>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The test suites in the XML do not appear to match xUnit.NET XML"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.DotNetxUnitParser{}.Parse(
				strings.NewReader(`<assemblies><assembly></assembly></assemblies>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match xUnit.NET XML"),
			)
			Expect(parseResult).To(BeNil())
		})
	})
})
