package parsing

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JavaScriptTestCafeParser struct{}

type JavaScriptTestCafeTest struct {
	Name           string          `json:"name"`
	Meta           json.RawMessage `json:"meta,omitempty"`
	Errs           []string        `json:"errs"`
	DurationMs     int             `json:"durationMs"`
	Unstable       bool            `json:"unstable"`
	ScreenshotPath *string         `json:"screenshotPath"`
	Skipped        bool            `json:"skipped,omitempty"`
}

type JavaScriptTestCafeFixture struct {
	Name  string                   `json:"name"`
	Path  string                   `json:"path"`
	Meta  json.RawMessage          `json:"meta,omitempty"`
	Tests []JavaScriptTestCafeTest `json:"tests"`
}

type JavaScriptTestCafeReport struct {
	StartTime  *time.Time                  `json:"startTime"`
	EndTime    *time.Time                  `json:"endTime"`
	UserAgents []string                    `json:"userAgents"`
	Passed     int                         `json:"passed"`
	Total      int                         `json:"total"`
	Skipped    int                         `json:"skipped"`
	Fixtures   []JavaScriptTestCafeFixture `json:"fixtures"`
	Warnings   []string                    `json:"warnings"`
}

func (p JavaScriptTestCafeParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var report JavaScriptTestCafeReport

	if err := json.NewDecoder(data).Decode(&report); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if report.StartTime == nil || report.UserAgents == nil || report.Fixtures == nil {
		return nil, errors.NewInputError("The JSON does not look like a TestCafe report")
	}

	otherErrors := make([]v1.OtherError, len(report.Warnings))
	for i, warning := range report.Warnings {
		otherErrors[i] = v1.OtherError{Message: warning}
	}

	tests := make([]v1.Test, 0)
	for _, fixture := range report.Fixtures {
		for _, test := range fixture.Tests {
			tests = append(tests, p.buildTest(report, fixture, test))
		}
	}

	return v1.NewTestResults(
		v1.JavaScriptTestCafeFramework,
		tests,
		otherErrors,
	), nil
}

func (p JavaScriptTestCafeParser) buildTest(
	report JavaScriptTestCafeReport,
	fixture JavaScriptTestCafeFixture,
	test JavaScriptTestCafeTest,
) v1.Test {
	lineage := []string{fixture.Name, test.Name}
	duration := time.Duration(test.DurationMs) * time.Millisecond

	var location *v1.Location
	if fixture.Path != "" {
		location = &v1.Location{File: fixture.Path}
	}

	meta := map[string]any{
		"unstable":   test.Unstable,
		"userAgents": report.UserAgents,
	}
	if test.ScreenshotPath != nil {
		meta["screenshotPath"] = *test.ScreenshotPath
	}
	if len(test.Meta) > 0 && string(test.Meta) != "null" {
		meta["meta"] = json.RawMessage(test.Meta)
	}
	if len(fixture.Meta) > 0 && string(fixture.Meta) != "null" {
		meta["fixtureMeta"] = json.RawMessage(fixture.Meta)
	}

	var status v1.TestStatus
	switch {
	case test.Skipped:
		status = v1.NewSkippedTestStatus(nil)
	case len(test.Errs) > 0:
		message := firstLine(test.Errs[0])
		backtrace := test.Errs
		status = v1.NewFailedTestStatus(&message, nil, backtrace)
	default:
		status = v1.NewSuccessfulTestStatus()
	}

	attempt := v1.TestAttempt{
		Duration:  &duration,
		Meta:      meta,
		Status:    status,
		StartedAt: report.StartTime,
	}

	return v1.Test{
		Name:     strings.Join(lineage, " "),
		Lineage:  lineage,
		Location: location,
		Attempt:  attempt,
	}
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
