package parsing

import (
	"encoding/json"
	"io"
	"regexp"
	"time"

	"github.com/mileusna/useragent"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// Parses https://github.com/douglasduteil/karma-json-reporter
type JavaScriptKarmaParser struct{}

type JavaScriptKarmaTestResults struct {
	Browsers map[string]*JavaScriptKarmaBrowser   `json:"browsers"`
	Result   map[string][]JavaScriptKarmaTestCase `json:"result"`
	Summary  *JavaScriptKarmaSummary              `json:"summary"`
}

type JavaScriptKarmaBrowser struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	Name     string `json:"name"`
	State    string `json:"state"`
}

type JavaScriptKarmaTestCase struct {
	FullName                  string                        `json:"fullName"`
	Description               string                        `json:"description"`
	ID                        string                        `json:"id"`
	Log                       []string                      `json:"log"`
	Skipped                   bool                          `json:"skipped"`
	Disabled                  bool                          `json:"disabled"`
	Pending                   bool                          `json:"pending"`
	Success                   bool                          `json:"success"`
	Suite                     []string                      `json:"suite"`
	Time                      int                           `json:"time"`
	ExecutedExpectationsCount int                           `json:"executedExpectationsCount"`
	PassedExpectations        []*JavaScriptKarmaExpectation `json:"passedExpectations"`
}

type JavaScriptKarmaExpectation struct {
	MatcherName string `json:"matcherName"`
	Message     string `json:"message"`
	Passed      bool   `json:"passed"`
	Stack       string `json:"stack"`
}

type JavaScriptKarmaSummary struct {
	Disconnected bool `json:"disconnected"`
	Error        bool `json:"error"`
	ExitCode     int  `json:"exitCode"`
	Failed       int  `json:"failed"`
	Skipped      int  `json:"skipped"`
	Success      int  `json:"success"`
}

var javaScriptKarmaBacktraceSeparatorRegexp = regexp.MustCompile(`\r?\n *`)

func (p JavaScriptKarmaParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults JavaScriptKarmaTestResults

	if err := json.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}

	if testResults.Browsers == nil {
		return nil, errors.NewInputError("The file does not look like a Karma file")
	}
	if testResults.Result == nil {
		return nil, errors.NewInputError("The file does not look like a Karma file")
	}
	if testResults.Summary == nil {
		return nil, errors.NewInputError("The file does not look like a Karma file")
	}

	tests := make([]v1.Test, 0)
	var currentBrowser *JavaScriptKarmaBrowser
	for browserID, testCases := range testResults.Result {
		currentBrowser = testResults.Browsers[browserID]
		if currentBrowser == nil {
			return nil, errors.NewInputError("The file does not look like a Karma file")
		}

		for _, testCase := range testCases {
			id := testCase.ID
			duration := time.Duration(testCase.Time * int(time.Millisecond))
			var status v1.TestStatus
			switch {
			case testCase.Skipped:
				status = v1.NewSkippedTestStatus(nil)
			case testCase.Pending:
				status = v1.NewPendedTestStatus(nil)
			case testCase.Disabled:
				status = v1.NewSkippedTestStatus(nil)
			case testCase.Success:
				status = v1.NewSuccessfulTestStatus()
			default:
				var errorMessage string
				var backtrace []string

				if len(testCase.Log) > 0 {
					firstLog := testCase.Log[0]
					firstLogLines := javaScriptKarmaBacktraceSeparatorRegexp.Split(firstLog, -1)
					if len(firstLogLines) > 0 {
						errorMessage = firstLogLines[0]
						backtrace = firstLogLines[1:]
					}
					status = v1.NewFailedTestStatus(&errorMessage, nil, backtrace)
				}
			}

			ua := useragent.Parse(currentBrowser.FullName)

			tests = append(
				tests,
				v1.Test{
					ID:      &id,
					Name:    testCase.FullName,
					Lineage: append(testCase.Suite, testCase.Description),
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Status:   status,
						Meta: map[string]any{
							"browserName":     ua.Name + " " + ua.OS,
							"browserFullName": currentBrowser.FullName,
							"browserId":       currentBrowser.ID,
						},
					},
				},
			)
		}
	}

	otherErrors := make([]v1.OtherError, 0)
	if testResults.Summary.Failed == 0 && testResults.Summary.Error {
		otherErrors = append(otherErrors, v1.OtherError{
			Message: "An unknown error occurred",
		})
	}

	return v1.NewTestResults(
		v1.JavaScriptKarmaFramework,
		tests,
		otherErrors,
	), nil
}
