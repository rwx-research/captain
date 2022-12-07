package parsing

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JavaScriptMochaParser struct{}

type JavaScriptMochaStats struct {
	Duration int    `json:"duration"`
	End      string `json:"end"`
	Failures int    `json:"failures"`
	Passes   *int   `json:"passes,omitempty"`
	Pending  int    `json:"pending"`
	Start    string `json:"start"`
	Suites   *int   `json:"suites,omitempty"`
	Tests    int    `json:"tests"`
}

type JavaScriptMochaError struct {
	Actual           *string `json:"actual,omitempty"`
	Code             *string `json:"code,omitempty"`
	Expected         *string `json:"expected,omitempty"`
	GeneratedMessage *bool   `json:"generatedMessage,omitempty"`
	Message          string  `json:"message"`
	Name             *string `json:"name,omitempty"`
	Operator         *string `json:"operator,omitempty"`
	Stack            string  `json:"stack"`
}

type JavaScriptMochaTest struct {
	Title        string                `json:"title"`
	FullTitle    string                `json:"fullTitle"`
	File         string                `json:"file"`
	Duration     int                   `json:"duration"`
	CurrentRetry int                   `json:"currentRetry"`
	Speed        *string               `json:"speed"`
	Err          *JavaScriptMochaError `json:"err"`
}

type JavaScriptMochaTestResults struct {
	Stats    *JavaScriptMochaStats `json:"stats"`
	Tests    []JavaScriptMochaTest `json:"tests"`
	Pending  []JavaScriptMochaTest `json:"pending"`
	Failures []JavaScriptMochaTest `json:"failures"`
	Passes   []JavaScriptMochaTest `json:"passes"`
}

var javaScriptMochaBacktraceSeparatorRegexp = regexp.MustCompile(`\r?\n\s{4}at`)

func (p JavaScriptMochaParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults JavaScriptMochaTestResults

	if err := json.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if testResults.Stats == nil || testResults.Stats.Passes == nil || testResults.Stats.Suites == nil {
		return nil, errors.NewInputError("No stats were found in the JSON")
	}
	if testResults.Tests == nil {
		return nil, errors.NewInputError("No tests were found in the JSON")
	}

	tests := make([]v1.Test, 0)
	for _, mochaTest := range testResults.Passes {
		tests = append(tests, p.successfulTestFrom(mochaTest))
	}
	for _, mochaTest := range testResults.Pending {
		tests = append(tests, p.pendedTestFrom(mochaTest))
	}
	for _, mochaTest := range testResults.Failures {
		tests = append(tests, p.failedTestFrom(mochaTest))
	}
	if len(tests) != len(testResults.Tests) {
		return nil, errors.NewInputError("The mocha JSON has inconsistently defined passes, pending, failures, and tests")
	}

	return v1.NewTestResults(
		v1.NewJavaScriptMochaFramework(),
		tests,
		nil,
	), nil
}

func (p JavaScriptMochaParser) successfulTestFrom(mochaTest JavaScriptMochaTest) v1.Test {
	test := p.testFrom(mochaTest)
	test.Attempt.Status = v1.NewSuccessfulTestStatus()
	return test
}

func (p JavaScriptMochaParser) failedTestFrom(mochaTest JavaScriptMochaTest) v1.Test {
	var message *string
	var exception *string
	var backtrace []string

	if mochaTest.Err != nil {
		message = &mochaTest.Err.Message
		exception = mochaTest.Err.Name

		stackParts := javaScriptMochaBacktraceSeparatorRegexp.Split(mochaTest.Err.Stack, -1)[1:]
		for _, part := range stackParts {
			backtrace = append(backtrace, fmt.Sprintf("at%s", part))
		}
	}

	test := p.testFrom(mochaTest)
	test.Attempt.Status = v1.NewFailedTestStatus(message, exception, backtrace)
	return test
}

func (p JavaScriptMochaParser) pendedTestFrom(mochaTest JavaScriptMochaTest) v1.Test {
	test := p.testFrom(mochaTest)
	test.Attempt.Status = v1.NewPendedTestStatus(nil)
	return test
}

func (p JavaScriptMochaParser) testFrom(mochaTest JavaScriptMochaTest) v1.Test {
	lineage := []string{strings.TrimSuffix(mochaTest.FullTitle, fmt.Sprintf(" %v", mochaTest.Title))}
	if mochaTest.FullTitle != mochaTest.Title {
		lineage = append(lineage, mochaTest.Title)
	}

	duration := time.Duration(mochaTest.Duration * int(time.Millisecond))

	pastAttempts := make([]v1.TestAttempt, mochaTest.CurrentRetry)
	for i := range pastAttempts {
		pastAttempts[i] = v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}
	}

	return v1.Test{
		Name:         mochaTest.FullTitle,
		Lineage:      lineage,
		Location:     &v1.Location{File: mochaTest.File},
		Attempt:      v1.TestAttempt{Duration: &duration},
		PastAttempts: pastAttempts,
	}
}
