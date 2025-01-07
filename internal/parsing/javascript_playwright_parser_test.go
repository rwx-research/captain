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

var _ = Describe("JavaScriptPlaywrightParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/playwright.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("sets the scope to the project", func() {
			fixture, err := os.Open("../../test/fixtures/playwright.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			test := testResults.Tests[0]
			Expect(*test.Scope).To(SatisfyAny(Equal("chromium"), Equal("firefox")))
		})

		It("parses the sample file with other errors", func() {
			fixture, err := os.Open("../../test/fixtures/playwright_with_other_errors.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses the sample file with an error in global setup", func() {
			fixture, err := os.Open("../../test/fixtures/playwright_global_other_error.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(strings.NewReader(`{`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like Playwright", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptPlaywrightParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The JSON does not look like a Playwright report"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptPlaywrightParser{}.Parse(strings.NewReader(
				`{"suites":[],"errors":[]}`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The JSON does not look like a Playwright report"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptPlaywrightParser{}.Parse(strings.NewReader(
				`{"config":{},"errors":[]}`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The JSON does not look like a Playwright report"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptPlaywrightParser{}.Parse(strings.NewReader(
				`{"config":{},"suites":[]}`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The JSON does not look like a Playwright report"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptPlaywrightParser{}.Parse(strings.NewReader(
				`{"config":{},"suites":[],"errors":[]}`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
		})
	})
})
