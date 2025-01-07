package parsing

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

type ElixirExUnitParser struct{}

type ElixirExUnitFailure struct {
	Message  *string `xml:"message,attr"`
	Contents *string `xml:",chardata"`
}

type ElixirExUnitSkipped struct {
	Message *string `xml:"message,attr"`
}

type ElixirExUnitTestCase struct {
	ClassName string               `xml:"classname,attr,omitempty"`
	Error     *ElixirExUnitFailure `xml:"error"`
	Failure   *ElixirExUnitFailure `xml:"failure"`
	Name      string               `xml:"name,attr"`
	Skipped   *ElixirExUnitSkipped `xml:"skipped"`
	Time      float64              `xml:"time,attr"`
	File      *string              `xml:"file,attr"`

	XMLName xml.Name `xml:"testcase"`
}

type ElixirExUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type ElixirExUnitTestSuite struct {
	Errors     int                    `xml:"errors,attr"`
	Failures   int                    `xml:"failures,attr"`
	Name       string                 `xml:"name,attr"`
	Skipped    int                    `xml:"skipped,attr"`
	TestCases  []ElixirExUnitTestCase `xml:"testcase"`
	Properties []ElixirExUnitProperty `xml:"properties>property"`
	Tests      *int                   `xml:"tests,attr"`
	Time       float64                `xml:"time,attr,omitempty"`

	XMLName xml.Name `xml:"testsuite"`
}

type ElixirExUnitTestResults struct {
	TestSuites []ElixirExUnitTestSuite `xml:"testsuite"`
	XMLName    xml.Name                `xml:"testsuites"`
}

var elixirExUnitNewlineRegexp = regexp.MustCompile(`\r?\n`)

func (p ElixirExUnitParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults ElixirExUnitTestResults

	if err := xml.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}
	if len(testResults.TestSuites) > 0 && testResults.TestSuites[0].Tests == nil {
		return nil, errors.NewInputError("The test suites in the XML do not appear to match ExUnit XML")
	}

	tests := make([]v1.Test, 0)
	for _, testSuite := range testResults.TestSuites {
		for _, testCase := range testSuite.TestCases {
			name := fmt.Sprintf("%v %v", testCase.ClassName, testCase.Name)
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

			fileAndLine := strings.Split(*testCase.File, ":")
			line, err := strconv.Atoi(fileAndLine[1])
			if err != nil {
				return nil, errors.NewInputError("Elixir ExUnit file attribute is improperly formatted; expected file:line")
			}
			location := v1.Location{File: fileAndLine[0], Line: &line}

			tests = append(
				tests,
				v1.Test{
					Name:     name,
					Location: &location,
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Status:   status,
					},
				},
			)
		}
	}

	return v1.NewTestResults(
		v1.ElixirExUnitFramework,
		tests,
		nil,
	), nil
}

func (p ElixirExUnitParser) NewFailedTestStatus(failure ElixirExUnitFailure) v1.TestStatus {
	failureMessage := failure.Message

	// The lines in here look something like:
	//
	// 1) test throws an exception (ExunitexampleWeb.ExceptionTest)
	// test/exunitexample_web/views/exception_test.exs:7
	// ** (throw) "some exception"
	// code: throw "some exception"
	// stacktrace:
	// 	test/exunitexample_web/views/exception_test.exs:8: (test)
	//
	// So, we'll search for the 'stacktrace:' and take the next line as the backtrace
	var lines []string
	if failure.Contents != nil {
		lines = elixirExUnitNewlineRegexp.Split(*failure.Contents, -1)
	}
	if len(lines) == 0 {
		return v1.NewFailedTestStatus(failureMessage, nil, nil)
	}

	var backtrace *string
	for i, line := range lines {
		if strings.Contains(line, "stacktrace:") && i+1 < len(lines) {
			trimmedBacktrace := strings.TrimSpace(lines[i+1])
			backtrace = &trimmedBacktrace
		}
	}
	if backtrace == nil {
		return v1.NewFailedTestStatus(failureMessage, nil, nil)
	}

	return v1.NewFailedTestStatus(failureMessage, nil, []string{*backtrace})
}
