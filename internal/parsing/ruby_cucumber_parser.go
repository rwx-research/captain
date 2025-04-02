package parsing

import (
	"encoding/json"
	"io"
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
	Match  RubyCucumberMatch   `json:"match"`
	Result *RubyCucumberResult `json:"result"`
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

func (p RubyCucumberParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var cucumberFeatures []RubyCucumberFeature

	if err := json.NewDecoder(data).Decode(&cucumberFeatures); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if len(cucumberFeatures) == 0 {
		return nil, errors.NewInputError("No test results were found in the JSON")
	}

	foundOneResult := false
outer:
	for _, feature := range cucumberFeatures {
		for _, element := range feature.Elements {
			for _, step := range element.Steps {
				if step.Result != nil {
					foundOneResult = true
					break outer
				}
			}

			for _, before := range element.Before {
				if before.Result != nil {
					foundOneResult = true
					break outer
				}
			}

			for _, after := range element.After {
				if after.Result != nil {
					foundOneResult = true
					break outer
				}
			}
		}
	}

	if !foundOneResult {
		return nil, errors.NewInputError("Found features, but no results in the JSON")
	}

	tests := make([]v1.Test, 0)
	otherErrors := make([]v1.OtherError, 0)

	// https://github.com/cucumber/cucumber-ruby/blob/d9d6f380c77b79c3670fa8f1d620d7b57f42b3ae/lib/cucumber/formatter/usage.rb#L141
	// failed, then skipped, then pending, then undefined, then passed
	priority := map[string]int{
		"failed":    5,
		"skipped":   4,
		"pending":   3,
		"undefined": 2,
		"passed":    1,
	}

	for _, feature := range cucumberFeatures {
		for i := 0; i < len(feature.Elements); i++ {
			element := feature.Elements[i]

			// Merge Background into the next Scenario
			if element.Type == "background" {
				if i+1 >= len(feature.Elements) || feature.Elements[i+1].Type != "scenario" {
					return nil, errors.NewInputError("Background must be followed by a Scenario in feature '%s'", feature.Name)
				}

				nextElement := &feature.Elements[i+1]

				nextElement.Steps = append(element.Steps, nextElement.Steps...)
				nextElement.Before = append(element.Before, nextElement.Before...)
				nextElement.After = append(element.After, nextElement.After...)
				continue
			}

			duration := time.Duration(0)
			allResults := make([]RubyCucumberResult, 0)

			for _, hook := range element.Before {
				if hook.Result.Duration != nil {
					duration += time.Duration(*hook.Result.Duration * int(time.Nanosecond))
				}

				if hook.Result == nil {
					break
				}

				allResults = append(allResults, *hook.Result)
			}

			for _, step := range element.Steps {
				if step.Result.Duration != nil {
					duration += time.Duration(*step.Result.Duration * int(time.Nanosecond))
				}

				if step.Result == nil {
					break
				}

				allResults = append(allResults, *step.Result)
			}

			for _, hook := range element.After {
				if hook.Result.Duration != nil {
					duration += time.Duration(*hook.Result.Duration * int(time.Nanosecond))
				}

				if hook.Result == nil {
					break
				}

				allResults = append(allResults, *hook.Result)
			}

			var stepStatus string
			var status v1.TestStatus

			for _, result := range allResults {
				if _, ok := priority[result.Status]; !ok {
					return nil, errors.NewInputError("Unexpected status %v", result.Status)
				}

				if stepStatus == "" || priority[stepStatus] < priority[result.Status] {
					stepStatus = result.Status
					switch result.Status {
					case "passed":
						status = v1.NewSuccessfulTestStatus()
					case "failed":
						status = v1.NewFailedTestStatus(result.ErrorMessage, nil, nil)
					case "skipped":
						status = v1.NewSkippedTestStatus(result.ErrorMessage)
					case "undefined":
						status = v1.NewTodoTestStatus(result.ErrorMessage)
					case "pending":
						status = v1.NewTodoTestStatus(result.ErrorMessage)
					default:
						return nil, errors.NewInputError("Unexpected status %v", result.Status)
					}
				}
			}

			location := v1.Location{File: feature.URI, Line: &element.Line}
			attempt := v1.TestAttempt{
				Duration: &duration,
				Status:   status,
				Meta:     map[string]any{"tags": element.Tags},
			}

			lineage := []string{feature.Name, element.Name}
			tests = append(
				tests,
				v1.Test{
					Name:         strings.Join(lineage, " > "),
					Lineage:      lineage,
					Location:     &location,
					Attempt:      attempt,
					PastAttempts: nil,
				},
			)
		}
	}

	return v1.NewTestResults(
		v1.RubyCucumberFramework,
		tests,
		otherErrors,
	), nil
}
