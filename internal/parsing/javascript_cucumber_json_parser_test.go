package parsing_test

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptCucumberJSONParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber-js.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptCucumberJSONParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.JavaScriptCucumberJSONParser{}.Parse(strings.NewReader(`{`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like Cucumber", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptCucumberJSONParser{}.Parse(strings.NewReader(`[]`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No test results were found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptCucumberJSONParser{}.Parse(strings.NewReader(`
				[
					{
						"id": "rule-sample",
						"uri": "features/rule.feature",
						"keyword": "Feature",
						"name": "Rule Sample",
						"description": "",
						"line": 2
					}
				]`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Found features, but no results in the JSON"))
			Expect(testResults).To(BeNil())
		})
	})
})
