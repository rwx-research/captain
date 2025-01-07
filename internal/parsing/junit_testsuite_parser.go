package parsing

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

type JUnitTestsuiteParser struct{}

func (p JUnitTestsuiteParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testSuite JUnitTestSuite

	if err := xml.NewDecoder(data).Decode(&testSuite); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}

	if testSuite.Tests == nil {
		return nil, errors.NewInputError("The test suite in the XML does not appear to match JUnit XML")
	}

	tests := make([]v1.Test, 0)
	var properties map[string]any
	if len(testSuite.Properties) > 0 {
		properties = make(map[string]any)
	}
	for _, property := range testSuite.Properties {
		properties[property.Name] = property.Value
	}

	for _, testCase := range testSuite.TestCases {
		// The a lot of reporter libraries allow switching these
		// We want the one that has the entire description (contains the short description)
		// e.g. classname="Some Tests with some context it passes" name="it passes"
		// We'd want the classname in the above case. Classname contains name, but name doesn't contain classname
		var name string
		switch {
		case strings.Contains(testCase.Name, testCase.ClassName) && strings.Contains(testCase.ClassName, testCase.Name):
			name = testCase.Name
		case !strings.Contains(testCase.Name, testCase.ClassName) && strings.Contains(testCase.ClassName, testCase.Name):
			name = testCase.ClassName
		case strings.Contains(testCase.Name, testCase.ClassName) && !strings.Contains(testCase.ClassName, testCase.Name):
			name = testCase.Name
		case !strings.Contains(testCase.Name, testCase.ClassName) && !strings.Contains(testCase.ClassName, testCase.Name):
			name = fmt.Sprintf("%s %s", testCase.ClassName, testCase.Name)
		default:
			return nil, errors.NewInternalError("Unreachable: reached default case of exhaustive switch statement")
		}

		duration := time.Duration(math.Round(testCase.Time * float64(time.Second)))

		var status v1.TestStatus
		switch {
		case testCase.Failure != nil:
			status = p.NewFailedTestStatus(*testCase.Failure)
		case testCase.Error != nil:
			status = p.NewFailedTestStatus(*testCase.Error)
		case testCase.Skipped != nil:
			status = v1.NewSkippedTestStatus(testCase.Skipped.Message)
		default:
			status = v1.NewSuccessfulTestStatus()
		}

		var location *v1.Location
		switch {
		case testCase.File != nil:
			location = &v1.Location{File: *testCase.File}
		case testSuite.File != nil:
			location = &v1.Location{File: *testSuite.File}
		default:
			location = nil
		}
		switch {
		case location != nil && testCase.Line != nil:
			location.Line = testCase.Line
		case location != nil && testCase.Lineno != nil:
			location.Line = testCase.Lineno
		default:
			// nothing to do here
		}

		tests = append(
			tests,
			v1.Test{
				Name:     name,
				Location: location,
				Attempt: v1.TestAttempt{
					Duration: &duration,
					Meta:     properties,
					Status:   status,
					Stderr:   testCase.SystemErr,
					Stdout:   testCase.SystemOut,
				},
			},
		)
	}

	return v1.NewTestResults(
		v1.NewOtherFramework(nil, nil),
		tests,
		nil,
	), nil
}

func (p JUnitTestsuiteParser) NewFailedTestStatus(failure JUnitFailure) v1.TestStatus {
	failureMessage := failure.Message
	failureException := failure.Type

	// The JUnit spec suggests this be used for a stack trace, so we'll assume such but it could be wrong.
	var lines []string
	switch {
	case failure.CDataContents != nil:
		lines = jUnitNewlineRegexp.Split(*failure.CDataContents, -1)
	case failure.ChardataContents != nil:
		lines = jUnitNewlineRegexp.Split(*failure.ChardataContents, -1)
	default:
		lines = nil
	}
	if len(lines) == 0 {
		return v1.NewFailedTestStatus(failureMessage, failureException, nil)
	}

	backtrace := make([]string, 0)
	for _, line := range lines {
		backtrace = append(backtrace, strings.TrimSpace(line))
	}
	return v1.NewFailedTestStatus(failureMessage, failureException, backtrace)
}
