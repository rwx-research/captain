package parsing

import (
	"encoding/json"
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JavaScriptJestParser struct{}

type JavaScriptJestSerializableError struct {
	Code    *int    `json:"code"`
	Message string  `json:"message"`
	Stack   *string `json:"stack"`
	Type    *string `json:"type"`
}

type JavaScriptJestUncheckedSnapshot struct {
	FilePath string   `json:"filePath"`
	Keys     []string `json:"keys"`
}

type JavaScriptJestSnapshot struct {
	Added               int                               `json:"added"`
	DidUpdate           bool                              `json:"didUpdate"`
	Failure             bool                              `json:"failure"`
	FilesAdded          int                               `json:"filesAdded"`
	FilesRemoved        int                               `json:"filesRemoved"`
	FilesRemovedList    []string                          `json:"filesRemovedList"`
	FilesUnmatched      int                               `json:"filesUnmatched"`
	FilesUpdated        int                               `json:"filesUpdated"`
	Matched             int                               `json:"matched"`
	Total               int                               `json:"total"`
	Unchecked           int                               `json:"unchecked"`
	UncheckedKeysByFile []JavaScriptJestUncheckedSnapshot `json:"uncheckedKeysByFile"`
	Unmatched           int                               `json:"unmatched"`
	Updated             int                               `json:"updated"`
}

type JavaScriptJestCallsite struct {
	Column int `json:"column"`
	Line   int `json:"line"`
}

// https://github.com/facebook/jest/blob/6fc1860a34ea64a7c3360580e2874c94a5c8fc83/packages/jest-test-result/src/types.ts#LL47
// https://github.com/facebook/jest/blob/6fc1860a34ea64a7c3360580e2874c94a5c8fc83/packages/jest-types/src/TestResult.ts#L16
// https://github.com/facebook/jest/blob/6fc1860a34ea64a7c3360580e2874c94a5c8fc83/packages/jest-types/src/TestResult.ts#LL8C15-L8C80
// Same deal here with things not in the spec, but in the output
type JavaScriptJestAssertionResult struct {
	AncestorTitles  []string                `json:"ancestorTitles"`
	Duration        *int                    `json:"duration"`
	FailureMessages []string                `json:"failureMessages"`
	FullName        string                  `json:"fullName"`
	Location        *JavaScriptJestCallsite `json:"location"`
	Status          string                  `json:"status"`
	Title           string                  `json:"title"`

	// Not in the spec
	Invocations       *int     `json:"invocations"`
	NumPassingAsserts *int     `json:"numPassingAsserts"`
	RetryReasons      []string `json:"retryReasons"`
}

// https://github.com/facebook/jest/blob/6fc1860a34ea64a7c3360580e2874c94a5c8fc83/packages/jest-test-result/src/types.ts#L124
type JavaScriptJestTestResult struct {
	AssertionResults []JavaScriptJestAssertionResult `json:"assertionResults"`
	EndTime          int                             `json:"endTime"`
	Message          string                          `json:"message"`
	Name             string                          `json:"name"`
	StartTime        int                             `json:"startTime"`
	Status           string                          `json:"status"`
	Summary          string                          `json:"summary"`
}

// Per the code, this is what we should expect to see:
// https://github.com/facebook/jest/blob/6fc1860a34ea64a7c3360580e2874c94a5c8fc83/packages/jest-test-result/src/types.ts#L135
// In actuality, the Jest code seems to dump extra details (TypeScript does structural typing, so an object that has at
// least the declared attributes will type check).
// Here, we adhere to the linked code, but optionally support the extra bits in case they decide to remove them later.
// Inline comments indicate which ones aren't in the spec. This is how they get formatted:
// https://github.com/facebook/jest/blob/6fc1860a34ea64a7c3360580e2874c94a5c8fc83/packages/jest-test-result/src/formatTestResults.ts#L17
type JavaScriptJestTestResults struct {
	NumFailedTests            int                        `json:"numFailedTests"`
	NumFailedTestSuites       int                        `json:"numFailedTestSuites"`
	NumPassedTests            int                        `json:"numPassedTests"`
	NumPassedTestSuites       int                        `json:"numPassedTestSuites"`
	NumPendingTests           int                        `json:"numPendingTests"`
	NumPendingTestSuites      int                        `json:"numPendingTestSuites"`
	NumRuntimeErrorTestSuites *int                       `json:"numRuntimeErrorTestSuites"`
	NumTotalTests             int                        `json:"numTotalTests"`
	NumTotalTestSuites        int                        `json:"numTotalTestSuites"`
	Snapshot                  *JavaScriptJestSnapshot    `json:"snapshot"`
	StartTime                 int                        `json:"startTime"`
	Success                   bool                       `json:"success"`
	TestResults               []JavaScriptJestTestResult `json:"testResults"`
	WasInterrupted            bool                       `json:"wasInterrupted"`

	// Not in the spec
	NumTodoTests *int                             `json:"numTodoTests"`
	OpenHandles  []int                            `json:"openHandles"`
	RunExecError *JavaScriptJestSerializableError `json:"runExecError"`
}

func (p JavaScriptJestParser) Parse(data io.Reader) (*ParseResult, error) {
	var testResults JavaScriptJestTestResults

	if err := json.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if testResults.TestResults == nil {
		return nil, errors.NewInputError("No test results were found in the JSON")
	}
	if testResults.Snapshot == nil {
		return nil, errors.NewInputError("No snapshot was found in the JSON")
	}
	if testResults.NumRuntimeErrorTestSuites == nil {
		return nil, errors.NewInputError("No number of runtime error test suites was found in the JSON")
	}
	if len(testResults.TestResults) > 0 && testResults.TestResults[0].AssertionResults == nil {
		return nil, errors.NewInputError("The test results in the JSON do not appear to match Jest JSON")
	}

	tests := make([]v1.Test, 0)
	otherErrors := make([]v1.OtherError, 0)
	return &ParseResult{
		Sentiment: PositiveParseResultSentiment,
		TestResults: v1.TestResults{
			Framework:   v1.NewJavaScriptJestFramework(),
			Summary:     v1.NewSummary(tests, otherErrors),
			Tests:       tests,
			OtherErrors: otherErrors,
		},
		Parser: p,
	}, nil
}
