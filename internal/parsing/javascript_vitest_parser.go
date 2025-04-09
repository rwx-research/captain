package parsing

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JavaScriptVitestParser struct{}

// https://github.com/vitest-dev/vitest/blob/95f0203f27f5659f5758638edc4d1d90283801ac/packages/snapshot/src/types/index.ts#L47-L50
type JavaScriptVitestUncheckedSnapshot struct {
	FilePath string   `json:"filePath"`
	Keys     []string `json:"keys"`
}

// https://github.com/vitest-dev/vitest/blob/95f0203f27f5659f5758638edc4d1d90283801ac/packages/snapshot/src/types/index.ts#L52-L67
type JavaScriptVitestSnapshot struct {
	Added               int                                 `json:"added"`
	DidUpdate           bool                                `json:"didUpdate"`
	Failure             bool                                `json:"failure"`
	FilesAdded          int                                 `json:"filesAdded"`
	FilesRemoved        int                                 `json:"filesRemoved"`
	FilesRemovedList    []string                            `json:"filesRemovedList"`
	FilesUnmatched      int                                 `json:"filesUnmatched"`
	FilesUpdated        int                                 `json:"filesUpdated"`
	Matched             int                                 `json:"matched"`
	Total               int                                 `json:"total"`
	Unchecked           int                                 `json:"unchecked"`
	UncheckedKeysByFile []JavaScriptVitestUncheckedSnapshot `json:"uncheckedKeysByFile"`
	Unmatched           int                                 `json:"unmatched"`
	Updated             int                                 `json:"updated"`
}

// https://github.com/vitest-dev/vitest/blob/95f0203f27f5659f5758638edc4d1d90283801ac/packages/vitest/src/node/reporters/json.ts#L16-L19
type JavaScriptVitestCallsite struct {
	Column int `json:"column"`
	Line   int `json:"line"`
}

// https://github.com/vitest-dev/vitest/blob/95f0203f27f5659f5758638edc4d1d90283801ac/packages/runner/src/types/tasks.ts#L105
type JavaScriptVitestTaskMeta struct{}

// https://github.com/vitest-dev/vitest/blob/95f0203f27f5659f5758638edc4d1d90283801ac/packages/vitest/src/node/reporters/json.ts#L30-L39
type JavaScriptVitestAssertionResult struct {
	AncestorTitles  []string                  `json:"ancestorTitles"`
	Duration        *float64                  `json:"duration"`
	FailureMessages []string                  `json:"failureMessages"`
	FullName        string                    `json:"fullName"`
	Location        *JavaScriptVitestCallsite `json:"location"`
	Status          string                    `json:"status"`
	Title           string                    `json:"title"`
	Meta            JavaScriptVitestTaskMeta  `json:"meta"`
}

// https://github.com/vitest-dev/vitest/blob/95f0203f27f5659f5758638edc4d1d90283801ac/packages/vitest/src/node/reporters/json.ts#L41-L50
type JavaScriptVitestTestResult struct {
	AssertionResults []JavaScriptVitestAssertionResult `json:"assertionResults"`
	EndTime          float64                           `json:"endTime"`
	Message          string                            `json:"message"`
	Name             string                            `json:"name"`
	StartTime        float64                           `json:"startTime"`
	Status           string                            `json:"status"`
}

// https://github.com/vitest-dev/vitest/blob/95f0203f27f5659f5758638edc4d1d90283801ac/packages/vitest/src/node/reporters/json.ts#L52-L69
type JavaScriptVitestTestResults struct {
	NumFailedTests       int                          `json:"numFailedTests"`
	NumFailedTestSuites  int                          `json:"numFailedTestSuites"`
	NumPassedTests       int                          `json:"numPassedTests"`
	NumPassedTestSuites  int                          `json:"numPassedTestSuites"`
	NumPendingTests      int                          `json:"numPendingTests"`
	NumPendingTestSuites int                          `json:"numPendingTestSuites"`
	NumTodoTests         int                          `json:"numTodoTests"`
	NumTotalTests        int                          `json:"numTotalTests"`
	NumTotalTestSuites   int                          `json:"numTotalTestSuites"`
	Snapshot             *JavaScriptVitestSnapshot    `json:"snapshot"`
	StartTime            float64                      `json:"startTime"`
	Success              bool                         `json:"success"`
	TestResults          []JavaScriptVitestTestResult `json:"testResults"`
}

var javaScriptVitestBacktraceSeparatorRegexp = regexp.MustCompile(`\r?\n\s{4}at`)

func (p JavaScriptVitestParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults JavaScriptVitestTestResults

	if err := json.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if testResults.TestResults == nil {
		return nil, errors.NewInputError("No test results were found in the JSON")
	}
	if testResults.Snapshot == nil {
		return nil, errors.NewInputError("No snapshot was found in the JSON")
	}

	tests := make([]v1.Test, 0)
	otherErrors := make([]v1.OtherError, 0)
	for _, testResult := range testResults.TestResults {
		sawFailedTest := false
		file := testResult.Name

		for _, assertionResult := range testResult.AssertionResults {
			lineage := assertionResult.AncestorTitles
			lineage = append(lineage, assertionResult.Title)
			name := strings.Join(lineage, " > ")

			var line *int
			var column *int
			if assertionResult.Location != nil {
				assertionResult := assertionResult
				line = &assertionResult.Location.Line
				column = &assertionResult.Location.Column
			}
			location := v1.Location{File: file, Line: line, Column: column}

			var duration *time.Duration
			if assertionResult.Duration != nil {
				transformedDuration := time.Duration(*assertionResult.Duration * float64(time.Millisecond))
				duration = &transformedDuration
			}

			var status v1.TestStatus
			switch assertionResult.Status {
			case "passed":
				status = v1.NewSuccessfulTestStatus()
			case "failed":
				message, backtrace := p.extractFailureMetadata(assertionResult.FailureMessages)
				status = v1.NewFailedTestStatus(message, nil, backtrace)
				sawFailedTest = true
			case "skipped":
				status = v1.NewSkippedTestStatus(nil)
			case "pending":
				status = v1.NewPendedTestStatus(nil)
			case "todo":
				status = v1.NewTodoTestStatus(nil)
			default:
				return nil, errors.NewInputError(
					"Unexpected status %q for assertion result %v",
					assertionResult.Status,
					assertionResult,
				)
			}

			attempt := v1.TestAttempt{Duration: duration, Status: status}
			tests = append(
				tests,
				v1.Test{
					Name:     name,
					Lineage:  lineage,
					Location: &location,
					Attempt:  attempt,
				},
			)
		}

		if !sawFailedTest && testResult.Status == "failed" {
			if len(testResult.Name) > 0 {
				otherErrors = append(otherErrors, v1.OtherError{
					Message:  testResult.Message,
					Location: &v1.Location{File: testResult.Name},
				})
			} else {
				otherErrors = append(otherErrors, v1.OtherError{Message: testResult.Message})
			}
		}
	}

	return v1.NewTestResults(
		v1.JavaScriptVitestFramework,
		tests,
		otherErrors,
	), nil
}

func (p JavaScriptVitestParser) extractFailureMetadata(failureMessages []string) (*string, []string) {
	var message *string
	var backtrace []string

	if failureMessages != nil && failureMessages[0] != "" {
		parts := javaScriptVitestBacktraceSeparatorRegexp.Split(failureMessages[0], -1)
		first, rest := parts[0], parts[1:]
		message = &first

		for _, part := range rest {
			backtrace = append(backtrace, fmt.Sprintf("at%s", part))
		}
	}

	return message, backtrace
}
