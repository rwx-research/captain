package parsing

import (
	"encoding/json"
	"io"

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
	Err          *JavaScriptMochaError `json:"err"`
}

type JavaScriptMochaTestResults struct {
	Stats    *JavaScriptMochaStats `json:"stats"`
	Tests    []JavaScriptMochaTest `json:"tests"`
	Pending  []JavaScriptMochaTest `json:"pending"`
	Failures []JavaScriptMochaTest `json:"failures"`
	Passes   []JavaScriptMochaTest `json:"passes"`
}

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

	return v1.NewTestResults(
		v1.NewJavaScriptMochaFramework(),
		tests,
		nil,
	), nil
}
