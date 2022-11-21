package parsing

import (
	"encoding/xml"
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// Parses https://github.com/michaelleeallen/mocha-junit-reporter
type JavaScriptCypressParser struct{}

type JavaScriptCypressFailure struct {
	Type     string `xml:"type,attr"`
	Message  string `xml:"message,attr"`
	Contents string `xml:",cdata"`
}

type JavaScriptCypressError struct {
	Type     string `xml:"type,attr"`
	Message  string `xml:"message,attr"`
	Contents string `xml:",cdata"`
}

type JavaScriptCypressSkipped struct{}

type JavaScriptCypressTestCase struct {
	ClassName string                    `xml:"classname,attr"`
	Name      string                    `xml:"name,attr"`
	Time      float64                   `xml:"time,attr"`
	Failure   *JavaScriptCypressFailure `xml:"failure"`
	Error     *JavaScriptCypressError   `xml:"error"`
	Skipped   *JavaScriptCypressSkipped `xml:"skipped"`
	SystemErr *string                   `xml:"system-err"`
	SystemOut *string                   `xml:"system-out"`
	XMLName   xml.Name                  `xml:"testcase"`
}

type JavaScriptCypressTestSuite struct {
	Errors    int                         `xml:"errors,attr"`
	Failures  int                         `xml:"failures,attr"`
	File      *string                     `xml:"file,attr"`
	Name      string                      `xml:"name,attr"`
	Skipped   int                         `xml:"skipped,attr"`
	TestCases []JavaScriptCypressTestCase `xml:"testcase"`
	Tests     int                         `xml:"tests,attr"`
	Time      float64                     `xml:"time,attr"`
	Timestamp string                      `xml:"timestamp,attr"`
	XMLName   xml.Name                    `xml:"testsuite"`
}

type JavaScriptCypressTestResults struct {
	Errors     int                          `xml:"errors,attr"`
	Failures   int                          `xml:"failures,attr"`
	Skipped    int                          `xml:"skipped,attr"`
	Tests      *int                         `xml:"tests,attr"`
	TestSuites []JavaScriptCypressTestSuite `xml:"testsuite"`
	XMLName    xml.Name                     `xml:"testsuites"`
}

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
	otherErrors := make([]v1.OtherError, 0)

	return &ParseResult{
		Sentiment: PositiveParseResultSentiment,
		TestResults: v1.TestResults{
			Framework:   v1.NewJavaScriptCypressFramework(),
			Summary:     v1.NewSummary(tests, otherErrors),
			Tests:       tests,
			OtherErrors: otherErrors,
		},
		Parser: p,
	}, nil
}
