package parsing

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type PHPUnitParser struct{}

type PHPUnitFailure struct {
	Type     *string `xml:"type,attr"`
	Contents *string `xml:",chardata"`
}

type PHPUnitSkipped struct{}

type PHPUnitTestCase struct {
	Class     string          `xml:"class,attr"`
	ClassName string          `xml:"classname,attr"`
	Error     *PHPUnitFailure `xml:"error"`
	Failure   *PHPUnitFailure `xml:"failure"`
	Name      string          `xml:"name,attr"`
	Skipped   *PHPUnitSkipped `xml:"skipped"`
	Time      float64         `xml:"time,attr"`
	File      string          `xml:"file,attr"`
	Line      int             `xml:"line,attr"`

	XMLName xml.Name `xml:"testcase"`
}

type PHPUnitTestSuite struct {
	Assertions int                `xml:"assertions,attr"`
	Errors     int                `xml:"errors,attr"`
	Failures   int                `xml:"failures,attr"`
	File       *string            `xml:"file,attr"`
	Name       string             `xml:"name,attr"`
	Skipped    int                `xml:"skipped,attr"`
	TestCases  []PHPUnitTestCase  `xml:"testcase"`
	Tests      *int               `xml:"tests,attr"`
	TestSuites []PHPUnitTestSuite `xml:"testsuite"`
	Time       float64            `xml:"time,attr"`
	Warnings   int                `xml:"warnings,attr"`

	XMLName xml.Name `xml:"testsuite"`
}

type PHPUnitTestResults struct {
	TestSuites []PHPUnitTestSuite `xml:"testsuite"`
	XMLName    xml.Name           `xml:"testsuites"`
}

var phpUnitNewlineRegexp = regexp.MustCompile(`\r?\n`)

func (p PHPUnitParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults PHPUnitTestResults

	if err := xml.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}

	tests := make([]v1.Test, 0)
	for _, suite := range testResults.TestSuites {
		foundTests, err := p.testsWithinSuite(suite)
		if err != nil {
			return nil, err
		}
		tests = append(tests, foundTests...)
	}

	return v1.NewTestResults(
		v1.PHPUnitFramework,
		tests,
		nil,
	), nil
}

func (p PHPUnitParser) testsWithinSuite(suite PHPUnitTestSuite) ([]v1.Test, error) {
	nestedTests := make([]v1.Test, 0)
	for _, nestedSuite := range suite.TestSuites {
		foundTests, err := p.testsWithinSuite(nestedSuite)
		if err != nil {
			return nil, err
		}
		nestedTests = append(nestedTests, foundTests...)
	}

	tests := make([]v1.Test, 0)
	for _, testCase := range suite.TestCases {
		name := fmt.Sprintf("%s::%s", testCase.Class, testCase.Name)
		lineage := []string{testCase.Class, testCase.Name}
		duration := time.Duration(math.Round(testCase.Time * float64(time.Second)))

		risky := false
		var status v1.TestStatus
		switch {
		case testCase.Failure != nil:
			determinedStatus, wasRisky := p.newFailedTestStatus(*testCase.Failure)

			risky = wasRisky
			if wasRisky {
				status = v1.NewSuccessfulTestStatus()
			} else {
				status = determinedStatus
			}
		case testCase.Error != nil:
			determinedStatus, wasRisky := p.newFailedTestStatus(*testCase.Error)

			risky = wasRisky
			if wasRisky {
				status = v1.NewSuccessfulTestStatus()
			} else {
				status = determinedStatus
			}
		case testCase.Skipped != nil:
			status = v1.NewSkippedTestStatus(nil)
		default:
			status = v1.NewSuccessfulTestStatus()
		}

		line := testCase.Line
		location := &v1.Location{File: testCase.File, Line: &line}

		tests = append(
			tests,
			v1.Test{
				Name:     name,
				Lineage:  lineage,
				Location: location,
				Attempt: v1.TestAttempt{
					Duration: &duration,
					Meta: map[string]any{
						"class": testCase.Class,
						"name":  testCase.Name,
						"risky": risky,
					},
					Status: status,
				},
			},
		)
	}

	tests = append(tests, nestedTests...)
	return tests, nil
}

// returns the determined failed status and whether it was risky or not
func (p PHPUnitParser) newFailedTestStatus(failure PHPUnitFailure) (v1.TestStatus, bool) {
	failureException := failure.Type
	risky := false

	if failureException != nil && *failureException == "PHPUnit\\Framework\\RiskyTestError" {
		risky = true
	}

	var lines []string
	if failure.Contents != nil {
		lines = phpUnitNewlineRegexp.Split(*failure.Contents, -1)
	}
	if len(lines) < 4 {
		return v1.NewFailedTestStatus(nil, failureException, nil), risky
	}

	message, backtracePart := lines[1], lines[3]

	return v1.NewFailedTestStatus(&message, failureException, []string{backtracePart}), risky
}
