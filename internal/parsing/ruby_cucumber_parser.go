package parsing

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type RubyCucumberParser struct{}

// relevant: the ruby implementation (our reference implementation) is more stringent than the schema
// schema is here: https://github.com/cucumber/cucumber-json-converter/blob/6281588f716cfbcf98b8e107e6f8650f5d92b53c/src/cucumber-ruby/RubyCucumberRubySchema.ts

// https://github.com/cucumber/cucumber-ruby/blob/b0c5eff4c3a6675f791692b677126e99788165b5/lib/cucumber/formatter/json.rb#L253-L265
// maps to a Lineage
type RubyCucumberFeature struct {
	ID          string                `json:"id"`
	URI         string                `json:"uri"`
	Keyword     string                `json:"keyword"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Line        int                   `json:"line"`
	Tags        []RubyCucumberTag     `json:"tags"`
	Elements    []RubyCucumberElement `json:"elements"`
}

type RubyCucumberTag struct {
	Name string `json:"name"`
	Line *int   `json:"line"`
}

// https://github.com/cucumber/cucumber-ruby/blob/b0c5eff4c3a6675f791692b677126e99788165b5/lib/cucumber/formatter/json.rb#L280-L288
// maps to a Lineage
type RubyCucumberElement struct {
	ID          *string            `json:"id"`
	Keyword     string             `json:"keyword"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Line        int                `json:"line"`
	Type        string             `json:"type"`
	Tags        []RubyCucumberTag  `json:"tags"`
	Steps       []RubyCucumberStep `json:"steps"`
	Before      []RubyCucumberHook `json:"before"`
	After       []RubyCucumberHook `json:"after"`
}

type RubyCucumberHook struct {
	Match  RubyCucumberMatch     `json:"match"`
	Result FailingCucumberResult `json:"result"`
}

// https://github.com/cucumber/cucumber-ruby/blob/b0c5eff4c3a6675f791692b677126e99788165b5/lib/cucumber/formatter/json.rb#L174-L178
// this is missing fields: doc_string, rows, src, mime_type, label
// see:
// https://github.com/cucumber/cucumber-ruby/blob/b0c5eff4c3a6675f791692b677126e99788165b5/lib/cucumber/formatter/json.rb#202
// maps to a Test
type RubyCucumberStep struct {
	Keyword string              `json:"keyword"`
	Name    string              `json:"name"`
	Line    int                 `json:"line"`
	Match   *RubyCucumberMatch  `json:"match"`
	Result  *RubyCucumberResult `json:"result"`
}

// https://github.com/cucumber/cucumber-ruby/blob/b0c5eff4c3a6675f791692b677126e99788165b5/lib/cucumber/formatter/json.rb#L213-L215
type RubyCucumberMatch struct {
	Location string `json:"location"`
}

// https://github.com/cucumber/cucumber-ruby/blob/b0c5eff4c3a6675f791692b677126e99788165b5/lib/cucumber/formatter/json.rb#L217-L224
type RubyCucumberResult struct {
	Status       string  `json:"status"`
	Duration     *int    `json:"duration"`
	ErrorMessage *string `json:"error_message"` // this is required if status is failed
}

type FailingCucumberResult struct {
	Status       string `json:"status"`
	Duration     *int   `json:"duration"`
	ErrorMessage string `json:"error_message"` // this is required if status is failed
}

func (p RubyCucumberParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var cucumberFeatures []RubyCucumberFeature

	if err := json.NewDecoder(data).Decode(&cucumberFeatures); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if len(cucumberFeatures) == 0 {
		return nil, errors.NewInputError("No test results were found in the JSON")
	}

	// at least one feature has an element
	foundOneStepResult := false
outer:
	for _, feature := range cucumberFeatures {
		// Check if the element is divisible by 2
		for _, element := range feature.Elements {
			for _, step := range element.Steps {
				if step.Result != nil {
					foundOneStepResult = true
					break outer
				}
			}
		}
	}

	if !foundOneStepResult {
		return nil, errors.NewInputError("Found features, but no steps in the JSON")
	}

	tests := make([]v1.Test, 0)
	otherErrors := make([]v1.OtherError, 0)

	for _, feature := range cucumberFeatures {
		for _, element := range feature.Elements {
			for _, hook := range element.Before {
				// do I reach this line?
				hookError := p.extractOtherError("before hook", hook)
				if hookError != nil {
					otherErrors = append(otherErrors, *hookError)
				}
			}

			for _, hook := range element.After {
				hookError := p.extractOtherError("after hook", hook)
				if hookError != nil {
					otherErrors = append(otherErrors, *hookError)
				}
			}

			for _, step := range element.Steps {
				if step.Result == nil {
					break // we can't do anything without a result
				}

				location := v1.Location{File: feature.URI, Line: &step.Line}

				var duration *time.Duration
				if step.Result.Duration != nil {
					transformedDuration := time.Duration(*step.Result.Duration * int(time.Nanosecond))
					duration = &transformedDuration
				}

				var status v1.TestStatus
				switch step.Result.Status {
				case "passed":
					status = v1.NewSuccessfulTestStatus()
				case "failed":
					status = v1.NewFailedTestStatus(step.Result.ErrorMessage, nil, nil)
				case "undefined":
					status = v1.NewTodoTestStatus(nil)
				case "skipped":
					status = v1.NewSkippedTestStatus(nil)
				case "pending":
					status = v1.NewPendedTestStatus(nil)
				default:
					return nil, errors.NewInputError(
						"Unexpected status %q for cucumber step %v",
						step.Result.Status,
						step,
					)
				}

				attempt := v1.TestAttempt{
					Duration: duration, Status: status,
					Meta: map[string]any{
						"tags":         element.Tags,
						"stepLocation": step.Match.Location,
						"elementStart": feature.URI + ":" + strconv.Itoa(element.Line),
					},
				}

				lineage := []string{feature.Name, element.Name, step.Name}
				// note: the location and id are different
				tests = append(
					tests,
					v1.Test{
						Name:         strings.Join(lineage, " > "),
						Lineage:      lineage,
						Location:     &location,
						Attempt:      attempt,
						PastAttempts: nil,
					})
			}

		}
	}

	return v1.NewTestResults(
		v1.RubyCucumberFramework,
		tests,
		otherErrors,
	), nil
}

func (p RubyCucumberParser) extractOtherError(metaType string, hook RubyCucumberHook) *v1.OtherError {
	if hook.Result.Status != "failed" {
		return nil
	}

	parts := strings.Split(hook.Match.Location, ":")
	var location v1.Location
	if len(parts) == 2 {
		lineString := parts[1]
		lineNumber, err := strconv.Atoi(lineString)
		if err == nil {
			location = v1.Location{File: parts[0], Line: &lineNumber}
		} else {
			location = v1.Location{File: parts[0]}
		}
	} else {
		location = v1.Location{File: hook.Match.Location}
	}

	return &v1.OtherError{
		Message:  hook.Result.ErrorMessage,
		Location: &location,
		Meta: map[string]any{
			"type": metaType,
		},
	}
}
