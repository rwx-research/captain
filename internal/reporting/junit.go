package reporting

import (
	"encoding/xml"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

func WriteJUnitSummary(file fs.File, testResults v1.TestResults, _ Configuration) error {
	result := parsing.JUnitTestResults{
		TestSuites: make([]parsing.JUnitTestSuite, 0),
	}

	finishedAt := time.Time{}
	startedAt := time.Time{}
	suite := parsing.JUnitTestSuite{}

	suite.Errors = testResults.Summary.OtherErrors + testResults.Summary.TimedOut
	suite.Failures = testResults.Summary.Canceled + testResults.Summary.Failed
	suite.Skipped = testResults.Summary.Pended + testResults.Summary.Skipped + testResults.Summary.Todo

	for _, test := range testResults.Tests {
		testCase := parsing.JUnitTestCase{
			Name:      test.Name,
			SystemErr: test.Attempt.Stderr,
			SystemOut: test.Attempt.Stdout,
		}

		if test.Attempt.Duration != nil {
			testCase.Time = test.Attempt.Duration.Seconds()
		}

		if test.Attempt.StartedAt != nil && test.Attempt.StartedAt.Before(startedAt) {
			startedAt = *test.Attempt.StartedAt
		}

		if test.Attempt.FinishedAt != nil && test.Attempt.FinishedAt.After(finishedAt) {
			finishedAt = *test.Attempt.FinishedAt
		}

		//nolint:exhaustive
		switch test.Attempt.Status.Kind {
		case v1.TestStatusPended, v1.TestStatusSkipped, v1.TestStatusTodo:
			testCase.Skipped = &parsing.JUnitSkipped{
				Message: test.Attempt.Status.Message,
			}
		case v1.TestStatusCanceled, v1.TestStatusFailed:
			testCase.Failure = &parsing.JUnitFailure{
				Message: test.Attempt.Status.Message,
			}
		case v1.TestStatusTimedOut:
			testCase.Error = &parsing.JUnitFailure{
				Message: test.Attempt.Status.Message,
			}
		}

		suite.TestCases = append(suite.TestCases, testCase)
	}

	if !startedAt.IsZero() {
		suite.Time = finishedAt.Sub(startedAt).Seconds()
		suite.Timestamp = startedAt.String()
	}

	totalTests := len(suite.TestCases)
	suite.Tests = &totalTests

	result.TestSuites = append(result.TestSuites, suite)

	_, err := file.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"))
	if err != nil {
		return errors.WithStack(err)
	}

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")

	if err := encoder.Encode(result); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
