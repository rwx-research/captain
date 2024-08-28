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

// Parses https://github.com/minitest-reporters/minitest-reporters/blob/73eea31b1e8b6af88c87f969cfa464d917f00cbb/lib/minitest/reporters/junit_reporter.rb#L16
type RubyMinitestParser struct{}

type RubyMinitestFailure struct {
	Contents *string `xml:",chardata"`
	Message  *string `xml:"message,attr"`
	Type     *string `xml:"type,attr"`
}

type RubyMinitestSkipped struct{}

type RubyMinitestTestCase struct {
	Assertions int                  `xml:"assertions,attr"`
	ClassName  string               `xml:"classname,attr"`
	Error      *RubyMinitestFailure `xml:"error"`
	Failure    *RubyMinitestFailure `xml:"failure"`
	File       string               `xml:"file,attr"`
	Lineno     int                  `xml:"lineno,attr"`
	Name       string               `xml:"name,attr"`
	Skipped    *RubyMinitestSkipped `xml:"skipped"`
	Time       float64              `xml:"time,attr"`

	XMLName xml.Name `xml:"testcase"`
}

type RubyMinitestTestSuite struct {
	Assertions int                    `xml:"assertions,attr"`
	Errors     int                    `xml:"errors,attr"`
	Failures   int                    `xml:"failures,attr"`
	Filepath   string                 `xml:"filepath,attr"`
	Name       string                 `xml:"name,attr"`
	Skipped    int                    `xml:"skipped,attr"`
	TestCases  []RubyMinitestTestCase `xml:"testcase"`
	Tests      *int                   `xml:"tests,attr"`
	Time       float64                `xml:"time,attr"`
	Timestamp  string                 `xml:"timestamp,attr"`

	XMLName xml.Name `xml:"testsuite"`
}

type RubyMinitestTestResults struct {
	TestSuites []RubyMinitestTestSuite `xml:"testsuite"`
	XMLName    xml.Name                `xml:"testsuites"`
}

var rubyMinitestFailureLocationRegexp = regexp.MustCompile(`\n.+\(.+\)\s\[(.+)\]:\n`)

func (p RubyMinitestParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults RubyMinitestTestResults

	if err := xml.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as XML: %s", err)
	}
	if len(testResults.TestSuites) == 0 || testResults.TestSuites[0].Tests == nil {
		return nil, errors.NewInputError("The test suites in the XML do not appear to match Ruby minitest XML")
	}

	tests := make([]v1.Test, 0)
	for _, testSuite := range testResults.TestSuites {
		for _, testCase := range testSuite.TestCases {
			duration := time.Duration(math.Round(testCase.Time * float64(time.Second)))
			lineage := []string{testCase.ClassName, testCase.Name}
			name := fmt.Sprintf("%s#%s", testCase.ClassName, testCase.Name)

			var status v1.TestStatus
			switch {
			case testCase.Failure != nil:
				status = p.NewFailedTestStatus(*testCase.Failure)
			case testCase.Error != nil:
				status = p.NewFailedTestStatus(*testCase.Error)
			case testCase.Skipped != nil:
				status = v1.NewSkippedTestStatus(nil)
			default:
				status = v1.NewSuccessfulTestStatus()
			}

			line := testCase.Lineno
			location := v1.Location{File: testCase.File, Line: &line}

			tests = append(
				tests,
				v1.Test{
					Name:     name,
					Lineage:  lineage,
					Location: &location,
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta:     map[string]any{"assertions": testCase.Assertions},
						Status:   status,
					},
				},
			)
		}
	}

	return v1.NewTestResults(
		v1.RubyMinitestFramework,
		tests,
		nil,
	), nil
}

var (
	rubyMinitestNewlineRegexp   = regexp.MustCompile(`\r?\n`)
	rubyMinitestBacktraceRegexp = regexp.MustCompile("\\s{4}.+:in `.+'")
)

func (p RubyMinitestParser) NewFailedTestStatus(failure RubyMinitestFailure) v1.TestStatus {
	failureMessage := failure.Message
	failureException := failure.Type

	if failure.Contents == nil {
		return v1.NewFailedTestStatus(failureMessage, failureException, nil)
	}

	lines := rubyMinitestNewlineRegexp.Split(strings.TrimSpace(*failure.Contents), -1)[2:]

	var failureBacktrace []string

	if len(lines) > 0 {
		failureMessageComponents := make([]string, 0)

		for _, line := range lines {
			if rubyMinitestBacktraceRegexp.Match([]byte(line)) {
				failureBacktrace = append(failureBacktrace, strings.TrimSpace(line))
			} else {
				failureMessageComponents = append(failureMessageComponents, line)
			}
		}

		constructedMessage := strings.Join(failureMessageComponents, "\n")
		failureMessage = &constructedMessage
	}

	if failureBacktrace == nil {
		location := rubyMinitestFailureLocationRegexp.FindStringSubmatch(*failure.Contents)
		if len(location) >= 2 {
			failureBacktrace = []string{location[1]}
		}
	}

	return v1.NewFailedTestStatus(failureMessage, failureException, failureBacktrace)
}
