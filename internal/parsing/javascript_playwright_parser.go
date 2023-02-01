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

type JavaScriptPlaywrightParser struct{}

// empty as we don't need anything from this that isn't already in the suite
// see https://github.com/microsoft/playwright/blob/e7088cc68573db2d7d83e2a184da16ba3f15a264/packages/playwright-test/types/testReporter.d.ts#L454-L467
type JavaScriptPlaywrightConfig struct{}

type JavaScriptPlaywrightTestError struct {
	Message *string `json:"message"`
	Stack   *string `json:"stack"`
	Value   *string `json:"value"`
}

type JavaScriptPlaywrightReportError struct {
	Message *string `json:"message"`
	Stack   *string `json:"stack"`
	Value   *string `json:"value"`
}

type JavaScriptPlaywrightAttachment struct {
	Name        string `json:"name"`
	Path        string `json:"path,omitempty"`
	Body        string `json:"body,omitempty"`
	ContentType string `json:"contentType"`
}

type JavaScriptPlaywrightStdIOEntry struct {
	Text   *string `json:"text,omitempty"`
	Buffer *string `json:"buffer,omitempty"`
}

type JavaScriptPlaywrightTestStep struct {
	Title    string                         `json:"title"`
	Duration int                            `json:"duration"` // milliseconds
	Error    *JavaScriptPlaywrightTestError `json:"error"`
	Steps    []JavaScriptPlaywrightTestStep `json:"steps,omitempty"`
}

type JavaScriptPlaywrightLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type JavaScriptPlaywrightTestResult struct {
	WorkerIndex int `json:"workerIndex"`
	// 'passed' | 'failed' | 'timedOut' | 'skipped' | 'interrupted'
	Status        string                            `json:"status"`
	Duration      int                               `json:"duration"` // milliseconds
	Error         *JavaScriptPlaywrightTestError    `json:"error"`
	Errors        []JavaScriptPlaywrightReportError `json:"errors"`
	Stdout        []JavaScriptPlaywrightStdIOEntry  `json:"stdout"`
	Stderr        []JavaScriptPlaywrightStdIOEntry  `json:"stderr"`
	Retry         int                               `json:"retry"`
	Steps         []JavaScriptPlaywrightTestStep    `json:"steps,omitempty"`
	StartTime     time.Time                         `json:"startTime"`
	Attachments   []JavaScriptPlaywrightAttachment  `json:"attachments"`
	ErrorLocation *JavaScriptPlaywrightLocation     `json:"errorLocation,omitempty"`
}

type JavaScriptPlaywrightAnnotation struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type JavaScriptPlaywrightTest struct {
	Timeout     int                              `json:"timeout"` // TODO(kkt) units
	Annotations []JavaScriptPlaywrightAnnotation `json:"annotations"`
	// 'passed' | 'failed' | 'timedOut' | 'skipped' | 'interrupted'
	ExpectedStatus string                           `json:"expectedStatus"`
	ProjectName    string                           `json:"projectName"`
	ProjectID      string                           `json:"projectId"`
	Results        []JavaScriptPlaywrightTestResult `json:"results"`
	// 'skipped' | 'expected' | 'unexpected' | 'flaky'
	Status string `json:"status"`
}

type JavaScriptPlaywrightSpec struct {
	Tags   []string                   `json:"tags"`
	Title  string                     `json:"title"`
	Ok     bool                       `json:"ok"`
	Tests  []JavaScriptPlaywrightTest `json:"tests"`
	ID     string                     `json:"id"`
	File   string                     `json:"file"`
	Line   int                        `json:"line"`
	Column int                        `json:"column"`
}

type JavaScriptPlaywrightSuite struct {
	Title  string                      `json:"title"`
	File   string                      `json:"file"`
	Column int                         `json:"column"`
	Line   int                         `json:"line"`
	Specs  []JavaScriptPlaywrightSpec  `json:"specs"`
	Suites []JavaScriptPlaywrightSuite `json:"suites,omitempty"`
}

type JavaScriptPlaywrightReport struct {
	Config *JavaScriptPlaywrightConfig     `json:"config"`
	Suites []JavaScriptPlaywrightSuite     `json:"suites"`
	Errors []JavaScriptPlaywrightTestError `json:"errors"`
}

var javaScriptPlaywrightBacktraceSeparatorRegexp = regexp.MustCompile(`\r?\n\s{4}at`)

func (p JavaScriptPlaywrightParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var report JavaScriptPlaywrightReport

	if err := json.NewDecoder(data).Decode(&report); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if report.Config == nil || report.Suites == nil || report.Errors == nil {
		return nil, errors.NewInputError("The JSON does not look like a Playwright report")
	}

	otherErrors := make([]v1.OtherError, len(report.Errors))
	for i, err := range report.Errors {
		if err.Value != nil {
			otherErrors[i] = v1.OtherError{Message: *err.Value}
			continue
		}

		if err.Message != nil && err.Stack != nil {
			stackParts := javaScriptPlaywrightBacktraceSeparatorRegexp.Split(*err.Stack, -1)[1:]
			backtrace := make([]string, len(stackParts))
			for i, part := range stackParts {
				backtrace[i] = fmt.Sprintf("at%s", part)
			}

			otherErrors[i] = v1.OtherError{
				Message:   *err.Message,
				Backtrace: backtrace,
			}
			continue
		}

		return nil, errors.NewInputError(
			"Unexpected error. Errors must have either a value _or_ a message and stack: %v",
			err,
		)
	}

	tests := make([]v1.Test, 0)
	for _, suite := range report.Suites {
		foundTests, err := p.testsWithinSuite(suite, []JavaScriptPlaywrightSuite{})
		if err != nil {
			return nil, err
		}
		tests = append(tests, foundTests...)
	}

	return v1.NewTestResults(
		v1.JavaScriptPlaywrightFramework,
		tests,
		otherErrors,
	), nil
}

func (p JavaScriptPlaywrightParser) testsWithinSuite(
	suite JavaScriptPlaywrightSuite,
	parents []JavaScriptPlaywrightSuite,
) ([]v1.Test, error) {
	nestedTests := make([]v1.Test, 0)
	nestedParents := make([]JavaScriptPlaywrightSuite, len(parents)+1)
	copy(nestedParents, parents)
	nestedParents[len(parents)] = suite

	for _, nestedSuite := range suite.Suites {
		mappedTests, err := p.testsWithinSuite(nestedSuite, nestedParents)
		if err != nil {
			return nil, err
		}
		nestedTests = append(nestedTests, mappedTests...)
	}

	tests := make([]v1.Test, 0)
	for _, spec := range suite.Specs {
		if len(spec.Tests) != 1 {
			// https://github.com/microsoft/playwright/blob/e7088cc68573db2d7d83e2a184da16ba3f15a264/packages/playwright-test/src/reporters/json.ts#L161
			return nil, errors.NewInputError(
				"Playwright specs must have exactly one test. Got: %v",
				spec.Tests,
			)
		}

		test := spec.Tests[0]

		lineage := make([]string, 0)
		for _, parent := range nestedParents {
			// We differentiate by file already in v1.Test.Location.File
			if parent.File == parent.Title {
				continue
			}

			lineage = append(lineage, parent.Title)
		}
		lineage = append(lineage, spec.Title)

		line := spec.Line
		column := spec.Column
		location := v1.Location{
			File:   spec.File,
			Line:   &line,
			Column: &column,
		}

		attempt := v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}
		pastAttempts := make([]v1.TestAttempt, 0)
		resultCount := len(test.Results)
		for i, result := range test.Results {
			duration := time.Duration(result.Duration * int(time.Millisecond))
			startedAt := result.StartTime

			stderrLines := make([]string, len(result.Stderr))
			for i, entry := range result.Stderr {
				if entry.Buffer != nil {
					stderrLines[i] = *entry.Buffer
				}
				if entry.Text != nil {
					stderrLines[i] = *entry.Text
				}
			}
			stderr := strings.Join(stderrLines, "")

			stdoutLines := make([]string, len(result.Stdout))
			for i, entry := range result.Stdout {
				if entry.Buffer != nil {
					stdoutLines[i] = *entry.Buffer
				}
				if entry.Text != nil {
					stdoutLines[i] = *entry.Text
				}
			}
			stdout := strings.Join(stdoutLines, "")

			var status v1.TestStatus
			switch result.Status {
			case "passed":
				status = v1.NewSuccessfulTestStatus()
			case "failed":
				var message *string
				var backtrace []string

				if result.Error != nil {
					message = result.Error.Message

					if result.Error.Stack != nil {
						stackParts := javaScriptPlaywrightBacktraceSeparatorRegexp.Split(*result.Error.Stack, -1)[1:]
						for _, part := range stackParts {
							backtrace = append(backtrace, fmt.Sprintf("at%s", part))
						}
					}
				}

				status = v1.NewFailedTestStatus(message, nil, backtrace)
			case "timedOut":
				status = v1.NewTimedOutTestStatus()
			case "skipped":
				status = v1.NewSkippedTestStatus(nil)
			case "interrupted":
				status = v1.NewCanceledTestStatus()
			default:
				return nil, errors.NewInputError("Unexpected test results status: %v", result.Status)
			}

			workingAttempt := v1.TestAttempt{
				Duration: &duration,
				Meta: map[string]any{
					"annotations": test.Annotations,
					"project":     test.ProjectName,
					"tags":        spec.Tags,
				},
				Status:    status,
				Stderr:    &stderr,
				Stdout:    &stdout,
				StartedAt: &startedAt,
			}

			if i == resultCount-1 {
				attempt = workingAttempt
			} else {
				pastAttempts = append(pastAttempts, workingAttempt)
			}
		}

		if test.ExpectedStatus == "failed" {
			if attempt.Status.Kind == v1.TestStatusFailed {
				attempt.Status = v1.NewSuccessfulTestStatus()
			} else {
				message := "Expected the test to fail, but it did not"
				attempt.Status = v1.NewFailedTestStatus(&message, nil, nil)
			}
		}

		tests = append(tests, v1.Test{
			Name:         strings.Join(lineage, " "),
			Lineage:      lineage,
			Location:     &location,
			Attempt:      attempt,
			PastAttempts: pastAttempts,
		})
	}

	tests = append(tests, nestedTests...)
	return tests, nil
}
