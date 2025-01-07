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

var _ = Describe("PythonPytestParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/pytest_reportlog.jsonl")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.PythonPytestParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.PythonPytestParser{}.Parse(strings.NewReader(`{"pytest_version":`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors JSON that doesn't look like pytest", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.PythonPytestParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Test results do not look like pytest resultlog (SessionStart report type missing)",
			))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.PythonPytestParser{}.Parse(strings.NewReader(`{"$report_type": "SessionStart"}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Test results do not look like pytest resultlog (Missing pytest version)",
			))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.PythonPytestParser{}.Parse(strings.NewReader(
				`
				{"pytest_version": "1.0.0", "$report_type": "SessionStart"}
				{"not valid"
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})
	})
})
