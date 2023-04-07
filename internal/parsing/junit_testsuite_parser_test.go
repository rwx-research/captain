package parsing_test

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JUnitTestsuiteParser", func() {
	Describe("Parse", func() {
		It("parses the sample testsuite file", func() {
			fixture, err := os.Open("../../test/fixtures/junit-no-testsuites-element.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("does not parse the sample testsuites file", func() {
			fixture, err := os.Open("../../test/fixtures/junit.xml")
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(fixture)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("errors on malformed XML", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("errors on XML that doesn't look like JUnit", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JUnitTestsuiteParser{}.Parse(
				strings.NewReader(`<testsuite></testsuite>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suite in the XML does not appear to match JUnit XML"),
			)
			Expect(testResults).To(BeNil())
		})

		It("extracts the file from a test case", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase
							name="some test name"
							classname="prefix some test name"
							file="some/path/to/file.js"
							line="12"
						>
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			line := 12
			Expect(*testResults.Tests[0].Location).To(Equal(
				v1.Location{File: "some/path/to/file.js", Line: &line},
			))
		})

		It("parses the duration as seconds", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase
							name="some test name"
							classname="prefix some test name"
							time="1.5249"
						>
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(*testResults.Tests[0].Attempt.Duration).To(Equal(
				time.Duration(1524900000),
			))
		})

		It("parses failures with inner CDATA", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase
							name="some test name"
							classname="prefix some test name"
							time="1.5249"
						>
							<failure type="someclass" message="some message"><![CDATA[line 1
								line 2

								line 3]]></failure>
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			message := "some message"
			exception := "someclass"
			backtrace := []string{
				"line 1",
				"line 2",
				"",
				"line 3",
			}
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewFailedTestStatus(&message, &exception, backtrace),
			))
		})

		It("parses failures with inner chardata", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase
							name="some test name"
							classname="prefix some test name"
							time="1.5249"
						>
							<failure type="someclass" message="some message">line 1
								line 2

								line 3</failure>
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			message := "some message"
			exception := "someclass"
			backtrace := []string{
				"line 1",
				"line 2",
				"",
				"line 3",
			}
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewFailedTestStatus(&message, &exception, backtrace),
			))
		})

		It("parses errors with inner CDATA", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase
							name="some test name"
							classname="prefix some test name"
							time="1.5249"
						>
							<error type="someclass" message="some message"><![CDATA[line 1
								line 2

								line 3]]></error>
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			message := "some message"
			exception := "someclass"
			backtrace := []string{
				"line 1",
				"line 2",
				"",
				"line 3",
			}
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewFailedTestStatus(&message, &exception, backtrace),
			))
		})

		It("parses errors with inner chardata", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase
							name="some test name"
							classname="prefix some test name"
							time="1.5249"
						>
							<error type="someclass" message="some message">line 1
								line 2

								line 3</error>
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			message := "some message"
			exception := "someclass"
			backtrace := []string{
				"line 1",
				"line 2",
				"",
				"line 3",
			}
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewFailedTestStatus(&message, &exception, backtrace),
			))
		})

		It("parses skipped tests with messages", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase
							name="some test name"
							classname="prefix some test name"
							time="1.5249"
						>
							<skipped message="some reason" />
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			message := "some reason"
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(
				v1.NewSkippedTestStatus(&message),
			))
		})

		It("calculates the correct name when the classname contains the name", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase name="some test name" classname="prefix some test name">
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name contains the classname", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase name="prefix some test name" classname="some test name">
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is the same as classname", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase name="prefix some test name" classname="prefix some test name">
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is entirely different from the classname", func() {
			testResults, err := parsing.JUnitTestsuiteParser{}.Parse(strings.NewReader(
				`
					<testsuite tests="1">
						<testcase name="some test name" classname="prefix">
						</testcase>
					</testsuite>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})
	})
})
