package parsing

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

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

var dotNetxUnitAssemblyNameRegexp = regexp.MustCompile(fmt.Sprintf(`[^%s]+$`, string(os.PathSeparator)))

var dotNetxUnitNewlineRegexp = regexp.MustCompile(`\r?\n`)

func (p DotNetxUnitParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults DotNetxUnitTestResults

	if err := xml.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}
	if len(testResults.Assemblies) > 0 && testResults.Assemblies[0].Collections == nil {
		return nil, errors.NewInputError("The test suites in the XML do not appear to match xUnit.NET XML")
	}

	tests := make([]v1.Test, 0)
	otherErrors := make([]v1.OtherError, 0)
	for _, assembly := range testResults.Assemblies {
		assemblyName := dotNetxUnitAssemblyNameRegexp.FindString(assembly.Name)

		for _, collection := range assembly.Collections {
			for _, testCase := range collection.Tests {
				duration := time.Duration(math.Round(testCase.Time * float64(time.Second)))

				meta := map[string]any{
					"assembly": assemblyName,
					"type":     testCase.Type,
					"method":   testCase.Method,
				}
				for _, trait := range testCase.Traits {
					meta[fmt.Sprintf("trait-%v", trait.Name)] = trait.Value
				}

				var location *v1.Location
				if testCase.SourceFile != nil {
					location = &v1.Location{File: *testCase.SourceFile, Line: testCase.SourceLine}
				}

				var stdout *string
				if testCase.Output != nil {
					stdout = &testCase.Output.Contents
				}

				var status v1.TestStatus
				switch testCase.Result {
				case "Pass":
					status = v1.NewSuccessfulTestStatus()
				case "Fail":
					var message *string
					var exception *string
					var backtrace []string
					if testCase.Failure != nil {
						exception = testCase.Failure.ExceptionType
						message, backtrace = p.FailureDetails(*testCase.Failure)
					}

					status = v1.NewFailedTestStatus(message, exception, backtrace)
				case "Skip":
					var message *string
					if testCase.SkippedReason != nil {
						message = &testCase.SkippedReason.Contents
					}

					status = v1.NewSkippedTestStatus(message)
				case "NotRun":
					status = v1.NewSkippedTestStatus(nil)
				default:
					return nil, errors.NewInputError("Unexpected result %q for test %v", testCase.Result, testCase)
				}

				tests = append(
					tests,
					v1.Test{
						ID:       testCase.ID,
						Name:     testCase.Name,
						Location: location,
						Attempt: v1.TestAttempt{
							Duration: &duration,
							Meta:     meta,
							Status:   status,
							Stdout:   stdout,
						},
					},
				)
			}
		}

		for _, assemblyError := range assembly.Errors {
			message, backtrace := p.FailureDetails(assemblyError.Failure)

			if message == nil {
				defaultMessage := fmt.Sprintf("An error occurred during %v", assemblyError.Type)
				message = &defaultMessage
			}

			otherErrors = append(otherErrors, v1.OtherError{
				Backtrace: backtrace,
				Exception: assemblyError.Failure.ExceptionType,
				Message:   *message,
				Meta: map[string]any{
					"assembly": assemblyName,
					"type":     assemblyError.Type,
				},
			})
		}
	}

	return &v1.TestResults{
		Framework:   v1.NewDotNetxUnitFramework(),
		Summary:     v1.NewSummary(tests, otherErrors),
		Tests:       tests,
		OtherErrors: otherErrors,
	}, nil
}

func (p DotNetxUnitParser) FailureDetails(failure DotNetxUnitFailure) (*string, []string) {
	var message *string
	if failure.Message != nil {
		message = &failure.Message.Contents
	}

	var backtrace []string
	if failure.Stacktrace != nil {
		backtrace = dotNetxUnitNewlineRegexp.Split(failure.Stacktrace.Contents, -1)
		for i, line := range backtrace {
			backtrace[i] = strings.TrimSpace(line)
		}
	}

	return message, backtrace
}
