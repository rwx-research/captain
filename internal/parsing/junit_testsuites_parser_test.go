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

var _ = Describe("JUnitTestsuitesParser", func() {
	Describe("Parse", func() {
		It("parses the sample testsuites file", func() {
			fixture, err := os.Open("../../test/fixtures/junit.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("does not parse the sample testsuite file", func() {
			fixture, err := os.Open("../../test/fixtures/junit-no-testsuites-element.xml")
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(fixture)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("parses empty files", func() {
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(`<testsuites></testsuites>`))
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults.Tests).To(HaveLen(0))
		})

		It("errors on malformed XML", func() {
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("errors on XML that doesn't look like JUnit", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("extracts the file from a test case", func() {
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								file="some/path/to/file.js"
								line="12"
							>
							</testcase>
						</testsuite>
					</testsuites>
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
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								time="1.5249"
							>
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(*testResults.Tests[0].Attempt.Duration).To(Equal(
				time.Duration(1524900000),
			))
		})

		It("parses failures with inner CDATA", func() {
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
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
					</testsuites>
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
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
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
					</testsuites>
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
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
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
					</testsuites>
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
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
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
					</testsuites>
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
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase
								name="some test name"
								classname="prefix some test name"
								time="1.5249"
							>
								<skipped message="some reason" />
							</testcase>
						</testsuite>
					</testsuites>
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
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase name="some test name" classname="prefix some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name contains the classname", func() {
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase name="prefix some test name" classname="some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is the same as classname", func() {
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase name="prefix some test name" classname="prefix some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is entirely different from the classname", func() {
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites>
						<testsuite tests="1">
							<testcase name="some test name" classname="prefix">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("parses nested testsuites (bun test format)", func() {
			testResults, err := parsing.JUnitTestsuitesParser{}.Parse(strings.NewReader(
				`
					<testsuites name="bun test" tests="3">
						<testsuite name="test.spec.ts" file="test.spec.ts" tests="3">
							<testsuite name="Suite" file="test.spec.ts" line="1" tests="3">
								<testsuite name="#method" file="test.spec.ts" line="2" tests="3">
									<testcase name="works" classname="#method > Suite" time="0.1" />
									<testsuite name="nested" file="test.spec.ts" line="4" tests="2">
										<testcase name="test 1" classname="nested > #method > Suite" time="0.2" />
										<testcase name="test 2" classname="nested > #method > Suite" time="0.3" />
									</testsuite>
								</testsuite>
							</testsuite>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests).To(HaveLen(3))
			Expect(testResults.Tests[0].Name).To(Equal("#method > Suite works"))
			Expect(testResults.Tests[1].Name).To(Equal("nested > #method > Suite test 1"))
			Expect(testResults.Tests[2].Name).To(Equal("nested > #method > Suite test 2"))
		})
	})
})
