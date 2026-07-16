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

// Parses the JUnit XML emitted by either minitest reporter:
//   - minitest-reporters: https://github.com/minitest-reporters/minitest-reporters/blob/73eea31b1e8b6af88c87f969cfa464d917f00cbb/lib/minitest/reporters/junit_reporter.rb#L16
//   - minitest-junit:     https://github.com/aespinosa/minitest-junit
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
	Line       int                  `xml:"line,attr"`
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

			// minitest-reporters emits the line number as `lineno`, minitest-junit as `line`.
			line := testCase.Lineno
			if line == 0 {
				line = testCase.Line
			}
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
	message := failure.Message
	var backtrace []string

	if failure.Contents != nil && rubyMinitestFailureLocationRegexp.MatchString(*failure.Contents) {
		// minitest-reporters: the detail block and backtrace both live in the body.
		if m, b := p.parseMinitestDetailBlock(*failure.Contents); m != nil {
			message, backtrace = m, b
		}
		if backtrace == nil {
			if loc := rubyMinitestFailureLocationRegexp.FindStringSubmatch(*failure.Contents); len(loc) >= 2 {
				backtrace = []string{loc[1]}
			}
		}
	} else {
		// minitest-junit: the detail block is in the `message` attribute and the body is a
		// single backtrace frame.
		if failure.Message != nil {
			if m, _ := p.parseMinitestDetailBlock(*failure.Message); m != nil {
				message = m
			}
		}
		if failure.Contents != nil {
			if frame := strings.TrimSpace(*failure.Contents); frame != "" {
				backtrace = []string{frame}
			}
		}
	}

	return v1.NewFailedTestStatus(message, failure.Type, backtrace)
}

// parseMinitestDetailBlock extracts the human-readable message and any backtrace frames from a
// minitest failure/error detail block, formatted as:
//
//	Failure:                        (or "Error:")
//	<identity> [<file>:<line>]:
//	<message line(s)>
//	<optional indented backtrace frame(s)>
//
// The two leading label/location lines are dropped so the stored message matches across the
// minitest-reporters and minitest-junit layouts. Returns a nil message when the block is too
// short to contain one, leaving the caller's fallback in place.
func (p RubyMinitestParser) parseMinitestDetailBlock(block string) (*string, []string) {
	lines := rubyMinitestNewlineRegexp.Split(strings.TrimSpace(block), -1)
	if len(lines) <= 2 {
		return nil, nil
	}
	lines = lines[2:]

	var failureBacktrace []string
	failureMessageComponents := make([]string, 0)

	for _, line := range lines {
		if rubyMinitestBacktraceRegexp.Match([]byte(line)) {
			failureBacktrace = append(failureBacktrace, strings.TrimSpace(line))
		} else {
			failureMessageComponents = append(failureMessageComponents, line)
		}
	}

	constructedMessage := strings.Join(failureMessageComponents, "\n")
	return &constructedMessage, failureBacktrace
}
