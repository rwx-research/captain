package parsing

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JUnitTestsuitesParser struct{}

type JUnitFailure struct {
	Type             *string `xml:"type,attr"`
	Message          *string `xml:"message,attr"`
	CDataContents    *string `xml:",cdata"`
	ChardataContents *string `xml:",chardata"`
}

type JUnitSkipped struct {
	Message *string `xml:"message,attr"`
}

type JUnitTestCase struct {
	ClassName string        `xml:"classname,attr,omitempty"`
	Error     *JUnitFailure `xml:"error"`
	Failure   *JUnitFailure `xml:"failure"`
	Name      string        `xml:"name,attr"`
	Skipped   *JUnitSkipped `xml:"skipped"`
	SystemErr *string       `xml:"system-err"`
	SystemOut *string       `xml:"system-out"`
	Time      float64       `xml:"time,attr"`

	// out of spec, but maybe interesting
	File   *string `xml:"file,attr"`
	Line   *int    `xml:"line,attr"`
	Lineno *int    `xml:"lineno,attr"`

	XMLName xml.Name `xml:"testcase"`
}

type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type JUnitTestSuite struct {
	Errors     int             `xml:"errors,attr"`
	Failures   int             `xml:"failures,attr"`
	Name       string          `xml:"name,attr"`
	Skipped    int             `xml:"skipped,attr"`
	TestCases  []JUnitTestCase `xml:"testcase"`
	Properties []JUnitProperty `xml:"properties>property"`
	Tests      *int            `xml:"tests,attr"`
	Time       float64         `xml:"time,attr,omitempty"`
	Timestamp  string          `xml:"timestamp,attr,omitempty"`

	// out of spec, but maybe interesting
	File *string `xml:"file,attr"`

	XMLName xml.Name `xml:"testsuite"`
}

type JUnitTestResults struct {
	TestSuites []JUnitTestSuite `xml:"testsuite"`
	XMLName    xml.Name         `xml:"testsuites"`
}

var jUnitNewlineRegexp = regexp.MustCompile(`\r?\n`)

func (p JUnitTestsuitesParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults JUnitTestResults

	if err := xml.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}

	if len(testResults.TestSuites) > 0 && testResults.TestSuites[0].Tests == nil {
		return nil, errors.NewInputError("The test suites in the XML do not appear to match JUnit XML")
	}

	tests := make([]v1.Test, 0)
	for _, testSuite := range testResults.TestSuites {
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
	}

	return v1.NewTestResults(
		v1.NewOtherFramework(nil, nil),
		tests,
		nil,
	), nil
}

func (p JUnitTestsuitesParser) NewFailedTestStatus(failure JUnitFailure) v1.TestStatus {
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
