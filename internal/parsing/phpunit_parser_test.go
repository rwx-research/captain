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

var _ = Describe("PHPUnitParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/phpunit.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.PHPUnitParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed XML", func() {
			testResults, err := parsing.PHPUnitParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("errors on XML that doesn't look like PHPUnit", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.PHPUnitParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.PHPUnitParser{}.Parse(strings.NewReader(`<testsuites></testsuites>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match PHPUnit XML"),
			)
			Expect(testResults).To(BeNil())

			testResults, err = parsing.PHPUnitParser{}.Parse(
				strings.NewReader(`<testsuites><testsuite></testsuite></testsuites>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match PHPUnit XML"),
			)
			Expect(testResults).To(BeNil())

			testResults, err = parsing.PHPUnitParser{}.Parse(
				strings.NewReader(`<testsuites><testsuite><testcase></testcase></testsuite></testsuites>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match PHPUnit XML"),
			)
			Expect(testResults).To(BeNil())
		})
	})
})
