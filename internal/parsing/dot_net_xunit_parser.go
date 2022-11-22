package parsing

import (
	"encoding/xml"
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type DotNetxUnitParser struct{}

type DotNetxUnitCdataContent struct {
	Contents string `xml:",cdata"`
}

type DotNetxUnitTrait struct {
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
	XMLName xml.Name `xml:"trait"`
}

type DotNetxUnitFailure struct {
	// children
	Message    *DotNetxUnitCdataContent `xml:"message"`
	Stacktrace *DotNetxUnitCdataContent `xml:"stack-trace"`

	// attributes
	ExceptionType *string `xml:"exception-type,attr"`

	XMLName xml.Name `xml:"failure"`
}

type DotNetxUnitTest struct {
	// children
	Traits        []DotNetxUnitTrait       `xml:"traits>trait"`
	Failure       *DotNetxUnitFailure      `xml:"failure"`
	Output        *DotNetxUnitCdataContent `xml:"output"`
	SkippedReason *DotNetxUnitCdataContent `xml:"reason"`

	// attributes
	ID         *string `xml:"id,attr"`
	Method     *string `xml:"method,attr"`
	Name       string  `xml:"name,attr"`
	Result     string  `xml:"result,attr"` // Pass, Fail, Skip, NotRun
	SourceFile *string `xml:"source-file,attr"`
	SourceLine *int    `xml:"source-line,attr"`
	Time       float64 `xml:"time,attr"`
	TimeRtf    *string `xml:"time-rtf,attr"`
	Type       *string `xml:"type,attr"`

	XMLName xml.Name `xml:"test"`
}

type DotNetxUnitCollection struct {
	// children
	Tests []DotNetxUnitTest `xml:"test"`

	// attributes
	ID      *string `xml:"id,attr"`
	Name    string  `xml:"name,attr"`
	Failed  int     `xml:"failed,attr"`
	NotRun  *int    `xml:"not-run,attr"`
	Passed  int     `xml:"passed,attr"`
	Skipped int     `xml:"skipped,attr"`
	Time    float64 `xml:"time,attr"`
	TimeRtf *string `xml:"time-rtf,attr"`
	Total   int     `xml:"total,attr"`

	XMLName xml.Name `xml:"collection"`
}

type DotNetxUnitError struct {
	// children
	Failure DotNetxUnitFailure `xml:"failure"`

	// attributes
	Name *string `xml:"name,attr"`
	Type string  `xml:"type,attr"`

	XMLName xml.Name `xml:"error"`
}

type DotNetxUnitAssembly struct {
	// children
	Collections []DotNetxUnitCollection `xml:"collection"`
	Errors      []DotNetxUnitError      `xml:"errors>error"`

	// attributes
	ConfigFile      *string `xml:"config-file,attr"`
	Environment     string  `xml:"environment,attr"`
	ErrorsCount     int     `xml:"errors,attr"`
	Failed          int     `xml:"failed,attr"`
	FinishRtf       *string `xml:"finish-rtf,attr"`
	ID              *string `xml:"id,attr"`
	Name            string  `xml:"name,attr"`
	NotRun          *int    `xml:"not-run,attr"`
	Passed          int     `xml:"passed,attr"`
	RunDate         string  `xml:"run-date,attr"`
	RunTime         string  `xml:"run-time,attr"`
	Skipped         int     `xml:"skipped,attr"`
	StartRtf        *string `xml:"start-rtf,attr"`
	TargetFramework *string `xml:"target-framework,attr"`
	TestFramework   string  `xml:"test-framework,attr"`
	Time            float64 `xml:"time,attr"`
	TimeRtf         *string `xml:"time-rtf,attr"`
	Total           int     `xml:"total,attr"`

	XMLName xml.Name `xml:"assembly"`
}

type DotNetxUnitTestResults struct {
	// children
	Assemblies []DotNetxUnitAssembly `xml:"assembly"`

	// attributes
	Computer      *string `xml:"computer,attr"`
	FinishRtf     *string `xml:"finish-rtf,attr"`
	ID            *string `xml:"id,attr"`
	SchemaVersion *int    `xml:"schema-version,attr"`
	StartRtf      *string `xml:"start-rtf,attr"`
	Timestamp     string  `xml:"timestamp,attr"`
	User          *string `xml:"user,attr"`

	XMLName xml.Name `xml:"assemblies"`
}

func (p DotNetxUnitParser) Parse(data io.Reader) (*ParseResult, error) {
	var testResults DotNetxUnitTestResults

	if err := xml.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}
	if len(testResults.Assemblies) < 1 || testResults.Assemblies[0].Collections == nil {
		return nil, errors.NewInputError("The test suites in the XML do not appear to match xUnit.NET XML")
	}

	tests := make([]v1.Test, 0)
	return &ParseResult{
		Sentiment: PositiveParseResultSentiment,
		TestResults: v1.TestResults{
			Framework: v1.NewDotNetxUnitFramework(),
			Summary:   v1.NewSummary(tests, nil),
			Tests:     tests,
		},
		Parser: p,
	}, nil
}
