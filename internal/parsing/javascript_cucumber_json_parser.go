package parsing

import (
	"encoding/json"
	"io"
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
				Status       string  `json:"status"`
				Duration     *int    `json:"duration"`
				ErrorMessage *string `json:"error_message"`
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
		}
	}

	if !foundOneResult {
		return nil, errors.NewInputError("Found features, but no results in the JSON")
	}

	tests := make([]v1.Test, 0)
	otherErrors := make([]v1.OtherError, 0)

	// https://github.com/cucumber/messages/blob/3a3c605d0ef7df53012d78ddc0796dbae3df17a1/javascript/src/getWorstTestStepResult.ts#L18-L26
	// failed, then ambiguous, then undefined, then pending, then skipped, then passed
	priority := map[string]int{
		"failed":    6,
		"ambiguous": 5,
		"undefined": 4,
		"pending":   3,
		"skipped":   2,
		"passed":    1,
	}

	for _, feature := range cucumberFeatures {
		for _, element := range feature.Elements {
			var stepStatus string
			var status v1.TestStatus
			duration := time.Duration(0)

			for _, step := range element.Steps {
				if step.Result.Duration != nil {
					duration += time.Duration(*step.Result.Duration * int(time.Nanosecond))
				}

				if step.Result == nil {
					break
				}

				if _, ok := priority[step.Result.Status]; !ok {
					return nil, errors.NewInputError(
						"Unexpected status %v for cucumber step %v",
						step.Result.Status,
						step,
					)
				}

				if stepStatus == "" || priority[stepStatus] < priority[step.Result.Status] {
					stepStatus = step.Result.Status
					switch step.Result.Status {
					case "passed":
						status = v1.NewSuccessfulTestStatus()
					case "failed":
						status = v1.NewFailedTestStatus(step.Result.ErrorMessage, nil, nil)
					case "skipped":
						status = v1.NewSkippedTestStatus(step.Result.ErrorMessage)
					case "undefined":
						status = v1.NewTodoTestStatus(step.Result.ErrorMessage)
					case "pending":
						status = v1.NewTodoTestStatus(step.Result.ErrorMessage)
					case "ambiguous":
						status = v1.NewFailedTestStatus(step.Result.ErrorMessage, nil, nil)
					default:
						return nil, errors.NewInputError(
							"Unexpected status %v for cucumber step %v",
							step.Result.Status,
							step,
						)
					}
				}
			}

			element := element
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
		v1.JavaScriptCucumberFramework,
		tests,
		otherErrors,
	), nil
}
