package parsing

import (
	"encoding/json"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

type RubyRSpecParser struct{}

type RubyRSpecException struct {
	Class     string   `json:"class"`
	Message   string   `json:"message"`
	Backtrace []string `json:"backtrace"`
}

type RubyRSpecExample struct {
	ID              *string             `json:"id"`
	Description     string              `json:"description"`
	FullDescription string              `json:"full_description"`
	Status          string              `json:"status"`
	FilePath        string              `json:"file_path"`
	LineNumber      int                 `json:"line_number"`
	RunTime         float64             `json:"run_time"`
	PendingMessage  *string             `json:"pending_message"`
	Exception       *RubyRSpecException `json:"exception"`
}

type RubyRSpecSummary struct {
	Duration                     float64 `json:"duration"`
	ExampleCount                 int     `json:"example_count"`
	FailureCount                 int     `json:"failure_count"`
	PendingCount                 int     `json:"pending_count"`
	ErrorsOutsideOfExamplesCount int     `json:"errors_outside_of_examples_count"`
}

type RubyRSpecTestResults struct {
	Version     *string            `json:"version"`
	Messages    []string           `json:"messages"`
	Examples    []RubyRSpecExample `json:"examples"`
	Summary     *RubyRSpecSummary  `json:"summary"`
	SummaryLine *string            `json:"summary_line"`
}

var fileRegexp = regexp.MustCompile(`\.rb(:.+|\[.+\])$`)

func (p RubyRSpecParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults RubyRSpecTestResults

	if err := json.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if testResults.Examples == nil {
		return nil, errors.NewInputError("No examples were found in the JSON")
	}
	if testResults.Summary == nil {
		return nil, errors.NewInputError("No summary was found in the JSON")
	}
	if len(testResults.Examples) > 0 && testResults.Examples[0].FullDescription == "" {
		return nil, errors.NewInputError("The examples in the JSON do not appear to match RSpec JSON")
	}

	tests := make([]v1.Test, 0)
	for _, example := range testResults.Examples {
		id := example.ID
		name := example.FullDescription

		lineage := []string{
			strings.TrimSpace(strings.TrimSuffix(example.FullDescription, example.Description)),
			example.Description,
		}

		file := example.FilePath
		if example.ID != nil {
			file = fileRegexp.ReplaceAllString(*example.ID, ".rb")
		}
		location := v1.Location{File: file}

		duration := time.Duration(math.Round(example.RunTime * float64(time.Second)))
		meta := make(map[string]any)

		if example.FilePath != "" {
			meta["filePath"] = example.FilePath
		}
		if example.LineNumber != 0 {
			meta["lineNumber"] = example.LineNumber
		}

		var status v1.TestStatus
		switch example.Status {
		case "failed":
			example := example
			status = v1.NewFailedTestStatus(
				&example.Exception.Message,
				&example.Exception.Class,
				example.Exception.Backtrace,
			)
		case "passed":
			status = v1.NewSuccessfulTestStatus()
		case "pending":
			status = v1.NewPendedTestStatus(example.PendingMessage)
		default:
			return nil, errors.NewInputError("Unexpected status %q for example %v", example.Status, example)
		}

		attempt := v1.TestAttempt{Duration: &duration, Meta: meta, Status: status}
		tests = append(
			tests,
			v1.Test{
				ID:       id,
				Name:     name,
				Lineage:  lineage,
				Location: &location,
				Attempt:  attempt,
			},
		)
	}

	otherErrors := make([]v1.OtherError, 0)
	if testResults.Summary.ErrorsOutsideOfExamplesCount > 0 {
		for i, message := range testResults.Messages {
			if i >= testResults.Summary.ErrorsOutsideOfExamplesCount {
				break
			}

			otherErrors = append(otherErrors, v1.OtherError{Message: message})
		}
	}

	return v1.NewTestResults(
		v1.RubyRSpecFramework,
		tests,
		otherErrors,
	), nil
}
