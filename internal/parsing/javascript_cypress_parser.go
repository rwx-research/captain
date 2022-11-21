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

// Parses https://github.com/michaelleeallen/mocha-junit-reporter
type JavaScriptCypressParser struct{}

type JavaScriptCypressFailure struct {
	Type     *string `xml:"type,attr"`
	Message  *string `xml:"message,attr"`
	Contents string  `xml:",cdata"`
}

type JavaScriptCypressSkipped struct{}

type JavaScriptCypressTestCase struct {
	ClassName string                    `xml:"classname,attr"`
	Error     *JavaScriptCypressFailure `xml:"error"`
	Failure   *JavaScriptCypressFailure `xml:"failure"`
	Name      string                    `xml:"name,attr"`
	Skipped   *JavaScriptCypressSkipped `xml:"skipped"`
	SystemErr *string                   `xml:"system-err"`
	SystemOut *string                   `xml:"system-out"`
	Time      float64                   `xml:"time,attr"`
	XMLName   xml.Name                  `xml:"testcase"`
}

type JavaScriptCypressProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type JavaScriptCypressTestSuite struct {
	Errors     int                         `xml:"errors,attr"`
	Failures   int                         `xml:"failures,attr"`
	File       *string                     `xml:"file,attr"`
	Name       string                      `xml:"name,attr"`
	Skipped    int                         `xml:"skipped,attr"`
	TestCases  []JavaScriptCypressTestCase `xml:"testcase"`
	Properties []JavaScriptCypressProperty `xml:"properties>property"`
	Tests      int                         `xml:"tests,attr"`
	Time       float64                     `xml:"time,attr"`
	Timestamp  string                      `xml:"timestamp,attr"`
	XMLName    xml.Name                    `xml:"testsuite"`
}

type JavaScriptCypressTestResults struct {
	Errors     int                          `xml:"errors,attr"`
	Failures   int                          `xml:"failures,attr"`
	Skipped    int                          `xml:"skipped,attr"`
	Tests      *int                         `xml:"tests,attr"`
	TestSuites []JavaScriptCypressTestSuite `xml:"testsuite"`
	XMLName    xml.Name                     `xml:"testsuites"`
}

var javaScriptCypressNewlineRegexp = regexp.MustCompile(`\r?\n`)

func (p JavaScriptCypressParser) Parse(data io.Reader) (*ParseResult, error) {
	var testResults JavaScriptCypressTestResults

	if err := xml.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}
	if testResults.Tests == nil {
		return nil, errors.NewInputError("No tests count was found in the XML")
	}
	if len(testResults.TestSuites) > 0 && testResults.TestSuites[0].File == nil {
		return nil, errors.NewInputError("The test suites in the XML do not appear to match Cypress XML")
	}

	tests := make([]v1.Test, 0)
	var currentFile *string
	for _, testSuite := range testResults.TestSuites {
		if testSuite.File != nil {
			currentFile = testSuite.File

			if !(strings.Contains(*currentFile, ".cy.") || strings.Contains(*currentFile, "cypress/")) {
				return nil, errors.NewInputError("The file does not look like a Cypress file: %q", *currentFile)
			}
		}

		var properties map[string]any
		if len(testSuite.Properties) > 0 {
			properties = make(map[string]any)
		}
		for _, property := range testSuite.Properties {
			properties[property.Name] = property.Value
		}

		for _, testCase := range testSuite.TestCases {
			// The mocha junit reporter library allows switching these
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
				failureMessage := testCase.Failure.Message
				failureException := testCase.Failure.Type

				lines := javaScriptCypressNewlineRegexp.Split(testCase.Failure.Contents, -1)
				var backtrace []string
				for _, line := range lines {
					if failureException != nil &&
						failureMessage != nil &&
						line == fmt.Sprintf("%s: %s", *failureException, *failureMessage) {
						continue
					}

					backtrace = append(backtrace, strings.TrimSpace(line))
				}

				status = v1.NewFailedTestStatus(failureMessage, failureException, backtrace)
			case testCase.Skipped != nil:
				status = v1.NewSkippedTestStatus(nil)
			case testCase.Error != nil:
				return nil, errors.NewInputError(
					"Unexpected <error> element in %q, mocha-junit-reporter does not emit these.",
					name,
				)
			default:
				status = v1.NewSuccessfulTestStatus()
			}

			var location *v1.Location
			if currentFile != nil {
				location = &v1.Location{File: *currentFile}
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

	return &ParseResult{
		Sentiment: PositiveParseResultSentiment,
		TestResults: v1.TestResults{
			Framework: v1.NewJavaScriptCypressFramework(),
			Summary:   v1.NewSummary(tests, nil),
			Tests:     tests,
		},
		Parser: p,
	}, nil
}
