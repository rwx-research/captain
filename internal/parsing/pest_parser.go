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

type PestParser struct{}

type PestFailure struct {
	Type     *string `xml:"type,attr"`
	Contents *string `xml:",chardata"`
}

type PestSkipped struct{}

type PestTestCase struct {
	Class     string       `xml:"class,attr"`
	ClassName string       `xml:"classname,attr"`
	Error     *PestFailure `xml:"error"`
	Failure   *PestFailure `xml:"failure"`
	Name      string       `xml:"name,attr"`
	Skipped   *PestSkipped `xml:"skipped"`
	Time      float64      `xml:"time,attr"`
	File      string       `xml:"file,attr"`
	Line      int          `xml:"line,attr"`

	XMLName xml.Name `xml:"testcase"`
}

type PestTestSuite struct {
	Assertions int             `xml:"assertions,attr"`
	Errors     int             `xml:"errors,attr"`
	Failures   int             `xml:"failures,attr"`
	File       *string         `xml:"file,attr"`
	Name       string          `xml:"name,attr"`
	Skipped    int             `xml:"skipped,attr"`
	TestCases  []PestTestCase  `xml:"testcase"`
	Tests      *int            `xml:"tests,attr"`
	TestSuites []PestTestSuite `xml:"testsuite"`
	Time       float64         `xml:"time,attr"`
	Warnings   int             `xml:"warnings,attr"`

	XMLName xml.Name `xml:"testsuite"`
}

type PestTestResults struct {
	TestSuites []PestTestSuite `xml:"testsuite"`
	XMLName    xml.Name        `xml:"testsuites"`
}

var pestNewlineRegexp = regexp.MustCompile(`\r?\n`)

// Pest retains non-ASCII bytes when generating PHP method names from descriptions.
var pestMethodNameRegexp = regexp.MustCompile(`[^a-zA-Z0-9_\x{0080}-\x{10FFFF}]`)

func (p PestParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults PestTestResults

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
		v1.PHPPestFramework,
		tests,
		nil,
	), nil
}

func (p PestParser) testsWithinSuite(suite PestTestSuite) ([]v1.Test, error) {
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
		file := testCase.File
		meta := map[string]any{
			"class": testCase.Class,
			"name":  testCase.Name,
			"risky": risky,
		}
		// Pest 3+ reports logical locations instead of physical filenames.
		if path, _, found := strings.Cut(file, ".php::"); found {
			file = path + ".php"
			if strings.HasPrefix(testCase.Name, suite.Name+" with data set ") {
				meta["pestDatasetName"] = suite.Name
			} else if method, ok := strings.CutPrefix(suite.Name, testCase.Class+"::__pest_evaluable_"); ok {
				// Early Pest 3 releases use the generated method name for dataset suites.
				for offset := 0; offset < len(testCase.Name); offset++ {
					if !strings.HasPrefix(testCase.Name[offset:], " with data set ") {
						continue
					}
					description := testCase.Name[:offset]
					generated := strings.ReplaceAll(strings.ReplaceAll(description, "_", "__"), " ", "_")
					if pestMethodNameRegexp.ReplaceAllString(generated, "_") == method {
						meta["pestDatasetName"] = description
						break
					}
				}
			}
		}
		location := &v1.Location{File: file, Line: &line}

		tests = append(
			tests,
			v1.Test{
				Name:     name,
				Lineage:  lineage,
				Location: location,
				Attempt: v1.TestAttempt{
					Duration: &duration,
					Meta:     meta,
					Status:   status,
				},
			},
		)
	}

	tests = append(tests, nestedTests...)
	return tests, nil
}

// returns the determined failed status and whether it was risky or not
func (p PestParser) newFailedTestStatus(failure PestFailure) (v1.TestStatus, bool) {
	failureException := failure.Type
	risky := failureException != nil && *failureException == "PHPUnit\\Framework\\RiskyTestError"

	var lines []string
	if failure.Contents != nil {
		lines = pestNewlineRegexp.Split(*failure.Contents, -1)
	}
	if len(lines) < 4 {
		return v1.NewFailedTestStatus(nil, failureException, nil), risky
	}

	message, backtracePart := lines[1], lines[3]

	return v1.NewFailedTestStatus(&message, failureException, []string{backtracePart}), risky
}
