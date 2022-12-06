package parsing_test

import (
	"os"
	"strings"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptMochaParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/mocha.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptMochaParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			Expect(testResults.Framework.Language).To(Equal(v1.FrameworkLanguageJavaScript))
			Expect(testResults.Framework.Kind).To(Equal(v1.FrameworkKindMocha))
			Expect(testResults.Summary.Tests).To(Equal(0))
			Expect(testResults.Summary.Successful).To(Equal(0))
			Expect(testResults.Summary.Failed).To(Equal(0))
			Expect(testResults.Summary.Pended).To(Equal(0))
			Expect(testResults.Summary.OtherErrors).To(Equal(0))
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.JavaScriptMochaParser{}.Parse(strings.NewReader(`{`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like Mocha", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptMochaParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No stats were found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptMochaParser{}.Parse(strings.NewReader(`{"stats": {}}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No stats were found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptMochaParser{}.Parse(
				strings.NewReader(`{ "stats": { "suites": 1, "passes": 1 } }`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No tests were found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptMochaParser{}.Parse(
				strings.NewReader(`{ "stats": { "suites": 1, "passes": 1 }, "tests": [] }`),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
		})
	})
})
