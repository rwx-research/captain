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

var _ = Describe("RubyCucumberParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/integration.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses the sample file that uses background", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/background_integration.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses a passing element", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/passing.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses a failing element", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/failing.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses a pending element", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/pending.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses a skipped element", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/skipped.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses a undefined element", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/undefined.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses an element with a failing before hook", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/before_fail.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses an element with a failing after hook", func() {
			fixture, err := os.Open("../../test/fixtures/cucumber/after_fail.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.RubyCucumberParser{}.Parse(strings.NewReader(`{`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like Cucumber", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.RubyCucumberParser{}.Parse(strings.NewReader(`[]`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No test results were found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.RubyCucumberParser{}.Parse(strings.NewReader(`
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
