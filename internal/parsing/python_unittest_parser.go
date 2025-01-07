package parsing

import (
	"encoding/xml"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

type PythonUnitTestParser struct{}

type PythonUnitTestFailure struct {
	Type          *string `xml:"type,attr"`
	Message       *string `xml:"message,attr"`
	CDataContents *string `xml:",cdata"`
}

type PythonUnitTestSkipped struct {
	Message *string `xml:"message,attr"`
	Type    *string `xml:"type,attr"`
}

type PythonUnitTestTestCase struct {
	ClassName string                 `xml:"classname,attr"`
	Error     *PythonUnitTestFailure `xml:"error"`
	Failure   *PythonUnitTestFailure `xml:"failure"`
	File      string                 `xml:"file,attr"`
	Line      int                    `xml:"line,attr"`
	Name      string                 `xml:"name,attr"`
	Skipped   *PythonUnitTestSkipped `xml:"skipped"`
	Time      float64                `xml:"time,attr"`
	Timestamp string                 `xml:"timestamp,attr"`

	XMLName xml.Name `xml:"testcase"`
}

type PythonUnitTestProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type PythonUnitTestTestSuite struct {
	Errors    int                      `xml:"errors,attr"`
	Failures  int                      `xml:"failures,attr"`
	File      string                   `xml:"file,attr"`
	Name      string                   `xml:"name,attr"`
	Skipped   int                      `xml:"skipped,attr"`
	TestCases []PythonUnitTestTestCase `xml:"testcase"`
	Tests     *int                     `xml:"tests,attr"`
	Time      float64                  `xml:"time,attr"`
	Timestamp string                   `xml:"timestamp,attr"`

	TestSuites *[]PythonUnitTestTestSuite `xml:"testsuite"`
}

var PythonUnitTestNewlineRegexp = regexp.MustCompile(`\r?\n`)

func (p PythonUnitTestParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults PythonUnitTestTestSuite

	if err := xml.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}

	var testSuites []PythonUnitTestTestSuite
	if testResults.TestSuites == nil {
		testSuites = []PythonUnitTestTestSuite{testResults}
	} else {
		testSuites = *testResults.TestSuites
	}

	if len(testSuites) > 0 && testSuites[0].Tests == nil {
		return nil, errors.NewInputError("The test suites in the XML do not appear to match unittest XML")
	}

	tests := make([]v1.Test, 0)
	for _, testSuite := range testSuites {
		for _, testCase := range testSuite.TestCases {
			name := testCase.ClassName + "." + testCase.Name
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

			testCase := testCase
			location := &v1.Location{File: testCase.File, Line: &testCase.Line}
			startedAt, err := time.Parse("2006-01-02T15:04:05", testCase.Timestamp)
			var finishedAt time.Time
			if err == nil {
				finishedAt = startedAt.Add(duration)
			}

			tests = append(
				tests,
				v1.Test{
					Name:     name,
					Location: location,
					Lineage:  []string{testCase.ClassName, testCase.Name},
					Attempt: v1.TestAttempt{
						Duration:   &duration,
						Status:     status,
						StartedAt:  &startedAt,
						FinishedAt: &finishedAt,
					},
				},
			)
		}
	}

	return v1.NewTestResults(
		v1.PythonUnitTestFramework,
		tests,
		nil,
	), nil
}

func (p PythonUnitTestParser) NewFailedTestStatus(failure PythonUnitTestFailure) v1.TestStatus {
	failureMessage := failure.Message
	failureException := failure.Type

	var lines []string
	switch {
	case failure.CDataContents != nil:
		lines = PythonUnitTestNewlineRegexp.Split(*failure.CDataContents, -1)
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
