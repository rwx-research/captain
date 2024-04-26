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

type JavaScriptCucumberJSONParser struct{}

type JavaScriptCucumberJSONFeature struct {
	URI         string `json:"uri"`
	ID          string `json:"id"`
	Keyword     string `json:"keyword"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Line        int    `json:"line"`
	Tags        []struct {
		Name string `json:"name"`
		Line *int   `json:"line"`
	} `json:"tags"`
	Elements []struct {
		ID          string `json:"id"`
		Keyword     string `json:"keyword"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Line        int    `json:"line"`
		Steps       []struct {
			Arguments []struct{} `json:"arguments"`
			Keyword   string     `json:"keyword"`
			Line      int        `json:"line"`
			Name      string     `json:"name"`
			Match     struct {
				Location string `json:"location"`
			} `json:"match"`
			Result *struct {
				Status       string `json:"status"`
				Duration     *int   `json:"duration"`
				ErrorMessage string `json:"error_message"`
			} `json:"result"`
		} `json:"steps"`
		Tags []struct {
			Name string `json:"name"`
			Line *int   `json:"line"`
		} `json:"tags"`
		Type string `json:"type"`
	} `json:"elements"`
}

func (p JavaScriptCucumberJSONParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var cucumberFeatures []JavaScriptCucumberJSONFeature

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
			for _, step := range element.Steps {
				if step.Result == nil {
					break // we can't do anything without a result
				}

				if step.Keyword == "Before" || step.Keyword == "After" {
					// treat hooks as other errors
					if step.Result.Status != "failed" {
						break
					}

					otherErrors = append(otherErrors, v1.OtherError{
						Message: step.Result.ErrorMessage,
						Meta: map[string]any{
							"type": step.Keyword,
						},
					})
					break
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
					status = v1.NewFailedTestStatus(&step.Result.ErrorMessage, nil, nil)
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
					Duration: duration,
					Status:   status,
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
		v1.JavaScriptCucumberFramework,
		tests,
		otherErrors,
	), nil
}
