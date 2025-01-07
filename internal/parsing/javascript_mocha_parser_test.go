package parsing_test

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

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
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
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

		It("errors when the Mocha JSON has inconsistent results", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptMochaParser{}.Parse(
				strings.NewReader(`
					{
						"stats": { "suites": 1, "passes": 1 },
						"tests": [{}, {}, {}, {}],
						"passes": [{}],
						"failures": [{}],
						"pending": [{}]
					}
				`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The mocha JSON has inconsistently defined passes, pending, failures, and tests"),
			)
			Expect(testResults).To(BeNil())
		})
	})
})
