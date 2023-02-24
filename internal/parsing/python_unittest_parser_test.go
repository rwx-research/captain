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

var _ = Describe("PythonUnitTestParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/unittest.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.PythonUnitTestParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("handles files with multiple test suites", func() {
			testResults, err := parsing.PythonUnitTestParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite name="test.TSF-20230222165213" tests="1">
							<testcase classname="test.TSF" name="test_choice" />
						</testsuite>
						<testsuite name="test.TS2F-20230222165214" tests="1">
							<testcase classname="test.TS2F" name="test_choice" />
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed XML", func() {
			testResults, err := parsing.PythonUnitTestParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("errors on XML that doesn't look like unittest", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.PythonUnitTestParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The test suites in the XML do not appear to match unittest XML"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.PythonUnitTestParser{}.Parse(
				strings.NewReader(`<testcase></testcase>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match unittest XML"),
			)
			Expect(testResults).To(BeNil())

			testResults, err = parsing.PythonUnitTestParser{}.Parse(
				strings.NewReader(`<testsuite><testcase></testcase></testsuite>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match unittest XML"),
			)
			Expect(testResults).To(BeNil())
		})
	})
})
