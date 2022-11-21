package parsing_test

import (
	"os"
	"strings"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptJestParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/jest.json")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.JavaScriptJestParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			Expect(parseResult.Parser).To(Equal(parsing.JavaScriptJestParser{}))
			Expect(parseResult.Sentiment).To(Equal(parsing.PositiveParseResultSentiment))
			Expect(parseResult.TestResults.Framework.Language).To(Equal(v1.FrameworkLanguageJavaScript))
			Expect(parseResult.TestResults.Framework.Kind).To(Equal(v1.FrameworkKindJest))
			Expect(parseResult.TestResults.Summary.Tests).To(Equal(0))
			Expect(parseResult.TestResults.Summary.Failed).To(Equal(0))
			Expect(parseResult.TestResults.Summary.Successful).To(Equal(0))
			Expect(parseResult.TestResults.Summary.OtherErrors).To(Equal(0))
		})

		It("errors on malformed JSON", func() {
			parseResult, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(`{`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(parseResult).To(BeNil())
		})

		It("errors on JSON that doesn't look like Jest", func() {
			var parseResult *parsing.ParseResult
			var err error

			parseResult, err = parsing.JavaScriptJestParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No test results were found in the JSON"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.JavaScriptJestParser{}.Parse(strings.NewReader(`{"testResults": []}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No snapshot was found in the JSON"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.JavaScriptJestParser{}.Parse(
				strings.NewReader(`{"testResults": [], "snapshot": {}}`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("No number of runtime error test suites was found in the JSON"),
			)
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.JavaScriptJestParser{}.Parse(
				strings.NewReader(`{"testResults": [{}], "numRuntimeErrorTestSuites": 0, "snapshot": {}}`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test results in the JSON do not appear to match Jest JSON"),
			)
			Expect(parseResult).To(BeNil())
		})
	})
})
