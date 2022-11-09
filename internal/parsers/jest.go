package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// Jest is a jest parser.
type Jest struct{}

type jestTestSuite struct {
	StartTime   *int64 `json:"startTime"`
	TestResults []struct {
		Name             string `json:"name"`
		AssertionResults []struct {
			AncestorTitles  []string `json:"ancestorTitles"`
			Duration        *int     `json:"duration"`
			FailureMessages []string `json:"failureMessages"`
			Status          string   `json:"status"`
			Title           string   `json:"title"`
		} `json:"assertionResults"`
	} `json:"testResults"`
}

// Parse attempts to parse the provided byte-stream as a Jest test suite.
func (j *Jest) Parse(content io.Reader) (map[string]testing.TestResult, error) {
	var testSuite jestTestSuite

	if err := json.NewDecoder(content).Decode(&testSuite); err != nil {
		return nil, errors.NewInputError("unable to parse document as JSON: %s", err)
	}

	if testSuite.StartTime == nil {
		return nil, errors.NewInputError("provided JSON document is not a Jest test results")
	}

	results := make(map[string]testing.TestResult)
	for _, testResult := range testSuite.TestResults {
		for _, assertionResult := range testResult.AssertionResults {
			var status testing.TestStatus
			var statusMessage string

			description := assertionResult.AncestorTitles
			description = append(description, assertionResult.Title)

			id := append([]string{testResult.Name}, description...)

			switch assertionResult.Status {
			case "passed":
				status = testing.TestStatusSuccessful
			case "failed":
				status = testing.TestStatusFailed
				statusMessage = strings.Join(assertionResult.FailureMessages, "\n\n")
			case "pending", "todo":
				status = testing.TestStatusPending
			}

			null := 0
			if assertionResult.Duration == nil {
				assertionResult.Duration = &null
			}

			results[strings.Join(id, " > ")] = testing.TestResult{
				Description:   strings.Join(description, " > "),
				Duration:      time.Duration(*assertionResult.Duration) * time.Second,
				Status:        status,
				StatusMessage: statusMessage,
			}
		}
	}

	return results, nil
}
