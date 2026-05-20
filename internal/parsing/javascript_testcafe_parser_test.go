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

var _ = Describe("JavaScriptTestCafeParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/testcafe.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptTestCafeParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses the sample file with fixture and test meta", func() {
			fixture, err := os.Open("../../test/fixtures/testcafe_with_meta.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptTestCafeParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.JavaScriptTestCafeParser{}.Parse(strings.NewReader(`{`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like TestCafe", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptTestCafeParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The JSON does not look like a TestCafe report"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptTestCafeParser{}.Parse(strings.NewReader(
				`{"userAgents":[],"fixtures":[]}`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The JSON does not look like a TestCafe report"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptTestCafeParser{}.Parse(strings.NewReader(
				`{"startTime":"1970-01-01T00:00:00.000Z","fixtures":[]}`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The JSON does not look like a TestCafe report"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptTestCafeParser{}.Parse(strings.NewReader(
				`{"startTime":"1970-01-01T00:00:00.000Z","userAgents":[]}`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The JSON does not look like a TestCafe report"))
			Expect(testResults).To(BeNil())
		})
	})
})
