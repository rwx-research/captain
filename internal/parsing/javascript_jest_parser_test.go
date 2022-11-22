package parsing_test

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptJestParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/jest.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptJestParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			Expect(testResults.Framework.Language).To(Equal(v1.FrameworkLanguageJavaScript))
			Expect(testResults.Framework.Kind).To(Equal(v1.FrameworkKindJest))
			Expect(testResults.Summary.Tests).To(Equal(18))
			Expect(testResults.Summary.Successful).To(Equal(6))
			Expect(testResults.Summary.Failed).To(Equal(6))
			Expect(testResults.Summary.Pended).To(Equal(3))
			Expect(testResults.Summary.Todo).To(Equal(3))
			Expect(testResults.Summary.OtherErrors).To(Equal(0))
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(`{`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like Jest", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptJestParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No test results were found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptJestParser{}.Parse(strings.NewReader(`{"testResults": []}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No snapshot was found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptJestParser{}.Parse(
				strings.NewReader(`{"testResults": [], "snapshot": {}}`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("No number of runtime error test suites was found in the JSON"),
			)
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptJestParser{}.Parse(
				strings.NewReader(`{"testResults": [{}], "numRuntimeErrorTestSuites": 0, "snapshot": {}}`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test results in the JSON do not appear to match Jest JSON"),
			)
			Expect(testResults).To(BeNil())
		})

		It("parses a minimally failing test", func() {
			zero := 0
			durationInMilliseconds := 653
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
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

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

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
			zero := 0
			durationInMilliseconds := 653
			invocations := 3
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
							{
								AncestorTitles:  []string{"part 1", "part 2"},
								Duration:        &durationInMilliseconds,
								FailureMessages: []string{"final\n\nmessage\n    at stacktrace"},
								Status:          "failed",
								Title:           "title",
								Location:        &parsing.JavaScriptJestCallsite{Line: 10, Column: 12},
								RetryReasons: []string{
									"first\n\nmessage\n    at stacktrace\n    at other trace",
									"second\n\nmessage",
								},
								Invocations: &invocations,
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			expectedDuration := time.Duration(653000000)
			line := 10
			column := 12
			firstFailureMessage := "first\n\nmessage"
			firstBacktrace := []string{"at stacktrace", "at other trace"}
			secondFailureMessage := "second\n\nmessage"
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
					PastAttempts: []v1.TestAttempt{
						{Status: v1.NewFailedTestStatus(&firstFailureMessage, nil, firstBacktrace)},
						{Status: v1.NewFailedTestStatus(&secondFailureMessage, nil, nil)},
					},
				},
			))
		})

		It("does not require retry reasons to capture retries (it can use invocations)", func() {
			zero := 0
			durationInMilliseconds := 653
			invocations := 3
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
							{
								AncestorTitles:  []string{"part 1", "part 2"},
								Duration:        &durationInMilliseconds,
								FailureMessages: []string{"final\n\nmessage\n    at stacktrace"},
								Status:          "failed",
								Title:           "title",
								Location:        &parsing.JavaScriptJestCallsite{Line: 10, Column: 12},
								Invocations:     &invocations,
							},
						},
					},
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

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
					PastAttempts: []v1.TestAttempt{
						{Status: v1.NewFailedTestStatus(nil, nil, nil)},
						{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
				},
			))
		})

		It("parses passed statuses", func() {
			zero := 0
			durationInMilliseconds := 653
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
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

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(v1.NewSuccessfulTestStatus()))
		})

		It("parses todo statuses", func() {
			zero := 0
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
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

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Attempt.Duration).To(BeNil())
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(v1.NewTodoTestStatus(nil)))
		})

		It("parses skipped statuses", func() {
			zero := 0
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
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

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(v1.NewSkippedTestStatus(nil)))
		})

		It("parses pending statuses", func() {
			zero := 0
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
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

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Attempt.Status).To(Equal(v1.NewPendedTestStatus(nil)))
		})

		It("errors on other statuses", func() {
			zero := 0
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
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

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unexpected status \"wat\""))
			Expect(testResults).To(BeNil())
		})

		It("parses an other error when a whole test result fails", func() {
			zero := 0
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:    "/some/path/to/name/of/file.js",
						Status:  "failed",
						Message: "the reason it failed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
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

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.OtherErrors[0]).To(Equal(v1.OtherError{Message: "the reason it failed"}))
			Expect(testResults.Tests[0]).NotTo(BeNil())
		})

		It("does not parse an other error when a whole test result fails due to a failing test", func() {
			zero := 0
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:    "/some/path/to/name/of/file.js",
						Status:  "failed",
						Message: "the reason it failed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
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

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.OtherErrors).To(HaveLen(0))
			Expect(testResults.Tests[0]).NotTo(BeNil())
		})

		It("parses an other error when there is a run exec error", func() {
			zero := 0
			stack := "the stacktrace\nwith multiple lines"
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
							{
								Status: "passed",
								Title:  "title",
							},
						},
					},
				},
				RunExecError: &parsing.JavaScriptJestSerializableError{
					Message: "the reason it failed",
					Stack:   &stack,
				},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.OtherErrors[0]).To(Equal(
				v1.OtherError{
					Message:   "the reason it failed",
					Backtrace: []string{"the stacktrace", "with multiple lines"},
				},
			))
			Expect(testResults.Tests[0]).NotTo(BeNil())
		})

		It("parses an other error when there are open handles", func() {
			zero := 0
			jestResults := parsing.JavaScriptJestTestResults{
				NumRuntimeErrorTestSuites: &zero,
				Snapshot:                  &parsing.JavaScriptJestSnapshot{},
				TestResults: []parsing.JavaScriptJestTestResult{
					{
						Name:   "/some/path/to/name/of/file.js",
						Status: "passed",
						AssertionResults: []parsing.JavaScriptJestAssertionResult{
							{
								Status: "passed",
								Title:  "title",
							},
						},
					},
				},
				OpenHandles: []any{struct{}{}},
			}
			data, err := json.Marshal(jestResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.JavaScriptJestParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.OtherErrors[0]).To(Equal(
				v1.OtherError{Message: "An open handle was detected"},
			))
			Expect(testResults.Tests[0]).NotTo(BeNil())
		})
	})
})
