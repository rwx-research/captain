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

var _ = Describe("JavaScriptVitestParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/vitest.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(`{`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like Vitest", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No test results were found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(`{"testResults": []}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No snapshot was found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptVitestParser{}.Parse(
				strings.NewReader(`{"testResults": [{}], "snapshot": {}}`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test results in the JSON do not appear to match Vitest JSON"),
			)
			Expect(testResults).To(BeNil())
		})

		It("parses a minimally failing test", func() {
			durationInMilliseconds := 653.0
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								Duration: &durationInMilliseconds,
								Status:   "failed",
								Title:    "title",
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			expectedDuration := time.Duration(653000000)
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					Name:     "title",
					Lineage:  []string{"title"},
					Location: &v1.Location{File: "/some/path/to/name/of/file.js"},
					Attempt: v1.TestAttempt{
						Duration: &expectedDuration,
						Status:   v1.NewFailedTestStatus(nil, nil, nil),
					},
				},
			))
		})

		It("parses a maximally failing test", func() {
			durationInMilliseconds := 653.0
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								AncestorTitles:  []string{"part 1", "part 2"},
								Duration:        &durationInMilliseconds,
								FailureMessages: []string{"final\n\nmessage\n    at stacktrace"},
								Status:          "failed",
								Title:           "title",
								Location:        &parsing.JavaScriptVitestCallsite{Line: 10, Column: 12},
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			expectedDuration := time.Duration(653000000)
			line := 10
			column := 12
			finalFailureMessage := "final\n\nmessage"
			finalBacktrace := []string{"at stacktrace"}
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					Name:     "part 1 > part 2 > title",
					Lineage:  []string{"part 1", "part 2", "title"},
					Location: &v1.Location{File: "/some/path/to/name/of/file.js", Line: &line, Column: &column},
					Attempt: v1.TestAttempt{
						Duration: &expectedDuration,
						Status:   v1.NewFailedTestStatus(&finalFailureMessage, nil, finalBacktrace),
					},
				},
			))
		})

		It("parses passed statuses", func() {
			durationInMilliseconds := 653.0
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								Duration: &durationInMilliseconds,
								Status:   "passed",
								Title:    "title",
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(v1.NewSuccessfulTestStatus()))
		})

		It("parses todo statuses", func() {
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								Status: "todo",
								Title:  "title",
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Attempt.Duration).To(BeNil())
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(v1.NewTodoTestStatus(nil)))
		})

		It("parses skipped statuses", func() {
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								Status: "skipped",
								Title:  "title",
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(v1.NewSkippedTestStatus(nil)))
		})

		It("parses pending statuses", func() {
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								Status: "pending",
								Title:  "title",
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(v1.NewPendedTestStatus(nil)))
		})

		It("errors on other statuses", func() {
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								Status: "wat",
								Title:  "title",
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unexpected status \"wat\""))
			Expect(testResults).To(BeNil())
		})

		It("parses an other error when a whole test result fails", func() {
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:    "/some/path/to/name/of/file.js",
						Status:  "failed",
						Message: "the reason it failed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								Status: "passed",
								Title:  "title",
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.OtherErrors[0]).To(Equal(v1.OtherError{Message: "the reason it failed"}))
			Expect(testResults.Tests[0]).NotTo(BeNil())
		})

		It("does not parse an other error when a whole test result fails due to a failing test", func() {
			jestResults := parsing.JavaScriptVitestTestResults{
				Snapshot: &parsing.JavaScriptVitestSnapshot{},
				TestResults: []parsing.JavaScriptVitestTestResult{
					{
						Name:    "/some/path/to/name/of/file.js",
						Status:  "failed",
						Message: "the reason it failed",
						AssertionResults: []parsing.JavaScriptVitestAssertionResult{
							{
								Status: "failed",
								Title:  "title",
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptVitestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.OtherErrors).To(HaveLen(0))
			Expect(testResults.Tests[0]).NotTo(BeNil())
		})
	})
})
