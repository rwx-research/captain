package parsers

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// Jest is a jest parser.
type Jest struct {
	result                      jestTestSuite
	resultIndex, assertionIndex int
}

type jestTestSuite struct {
	StartTime   *int64 `json:"startTime"`
	TestResults []struct {
		Name             string           `json:"name"`
		AssertionResults []jestTestResult `json:"assertionResults"`
	} `json:"testResults"`
}

type jestTestResult struct {
	AncestorTitles []string `json:"ancestorTitles"`
	Title          string   `json:"title"`
	Status         string   `json:"status"`
}

// Parse attempts to parse the provided byte-stream as a Jest test suite.
func (j *Jest) Parse(content io.Reader) error {
	if err := json.NewDecoder(content).Decode(&j.result); err != nil {
		return errors.NewInputError("unable to parse document as JSON: %s", err)
	}

	if j.result.StartTime == nil {
		return errors.NewInputError("provided JSON document is not a Jest artifact")
	}

	return nil
}

func (j *Jest) testResult() jestTestResult {
	return j.result.TestResults[j.resultIndex].AssertionResults[j.assertionIndex-1]
}

// IsTestCaseFailed returns whether or not the current test case has failed.
//
// Caution: this method will panic unless `NextTestCase` was previously called.
func (j *Jest) IsTestCaseFailed() bool {
	return j.testResult().Status == "failed"
}

// NextTestCase prepares the next test case for reading. It returns 'true' if this was successful and 'false' if there
// is no further test case to process.
//
// Caution: This method needs to be called before any other further data is read from the parser.
func (j *Jest) NextTestCase() bool {
	j.assertionIndex++

	if len(j.result.TestResults) == 0 {
		return false
	}

	if j.assertionIndex > len(j.result.TestResults[j.resultIndex].AssertionResults) {
		j.assertionIndex = 0
		j.resultIndex++

		if j.resultIndex == len(j.result.TestResults) {
			return false
		}

		return j.NextTestCase()
	}

	return true
}

// TestCaseID returns the ID of the current test case.
//
// Caution: this method will panic unless `NextTestCase` was previously called.
func (j *Jest) TestCaseID() string {
	assertionName := strings.Join(append(j.testResult().AncestorTitles, j.testResult().Title), " > ")
	return fmt.Sprintf("%s > %s", j.result.TestResults[j.resultIndex].Name, assertionName)
}
