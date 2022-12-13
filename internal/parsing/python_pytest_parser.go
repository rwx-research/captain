package parsing

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type PythonPytestParser struct{}

type PythonPytestSessionStart struct {
	PytestVersion *string `json:"pytest_version"`
	ReportType    *string `json:"$report_type"`
}

// This is a subset of the failed longrepr because it's quite challenging to model in Go
type PythonPytestFailedLongrepr struct {
	Reprcrash struct {
		Path    string `json:"path"`
		Lineno  int    `json:"lineno"`
		Message string `json:"message"`
	} `json:"reprcrash"`
}

type PythonPytestSkippedLongrepr []any

type PythonPytestTestResult struct {
	Nodeid         string          `json:"nodeid"`
	Location       []any           `json:"location"` // ["test_top_level.py", 0, "test_top_level_passing"],
	Keywords       map[string]int  `json:"keywords"`
	Outcome        string          `json:"outcome"` // failed, passed, skipped
	Longrepr       json.RawMessage `json:"longrepr"`
	When           string          `json:"when"`
	UserProperties [][]any         `json:"user_properties"`
	Sections       []any           `json:"sections"` // unsure how this is used, moving on for now
	Duration       float64         `json:"duration"` // in seconds
	ReportType     string          `json:"$report_type"`

	// only when run in parallel w/ pytest-xdist
	ItemIndex  int    `json:"item_index"`
	WorkerID   string `json:"worker_id"`
	TestrunUID string `json:"testrun_uid"`
	Node       string `json:"node"`
}

func (p PythonPytestParser) Parse(data io.Reader) (*v1.TestResults, error) {
	decoder := json.NewDecoder(data)

	var sessionStart PythonPytestSessionStart
	if err := decoder.Decode(&sessionStart); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if sessionStart.ReportType == nil || *sessionStart.ReportType != "SessionStart" {
		return nil, errors.NewInputError(
			"Test results do not look like pytest resultlog (SessionStart report type missing)",
		)
	}
	if sessionStart.PytestVersion == nil {
		return nil, errors.NewInputError(
			"Test results do not look like pytest resultlog (Missing pytest version)",
		)
	}

	testsByNodeid := map[string]v1.Test{}
	for decoder.More() {
		var testResult PythonPytestTestResult
		if err := decoder.Decode(&testResult); err != nil {
			return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
		}
		if testResult.ReportType != "TestReport" {
			continue
		}

		file := testResult.Location[0].(string)
		line := int(testResult.Location[1].(float64))
		location := v1.Location{
			File: file,
			Line: &line,
		}
		duration := time.Duration(math.Round(testResult.Duration * float64(time.Second)))
		status, err := p.statusOf(testResult)
		if err != nil {
			return nil, err
		}
		userProperties := map[string]any{}
		for _, userProperty := range testResult.UserProperties {
			key, value := userProperty[0], userProperty[1]
			userProperties[key.(string)] = value
		}

		test, ok := testsByNodeid[testResult.Nodeid]
		if ok {
			newDuration := *test.Attempt.Duration + duration
			test.Attempt.Duration = &newDuration
			test.Attempt.Meta = map[string]any{"user_properties": userProperties}

			// when the test is skipped, the setup is not run and the teardown is always passing, so we can
			// keep the original status
			if test.Attempt.Status.Kind == v1.TestStatusSkipped {
				continue
			}

			// when the test has failed, we don't want to overwrite it with a success
			if test.Attempt.Status.Kind == v1.TestStatusFailed {
				continue
			}

			test.Attempt.Status = *status
		} else {
			name := strings.TrimPrefix(testResult.Nodeid, fmt.Sprintf("%v::", location.File))
			test = v1.Test{
				ID:       &testResult.Nodeid,
				Name:     name,
				Lineage:  strings.Split(name, "::"),
				Location: &location,
				Attempt: v1.TestAttempt{
					Duration: &duration,
					Status:   *status,
					Meta:     map[string]any{"user_properties": userProperties},
				},
			}
		}
		testsByNodeid[testResult.Nodeid] = test
	}

	tests := make([]v1.Test, len(testsByNodeid))
	i := 0
	for _, test := range testsByNodeid {
		tests[i] = test
		i++
	}

	// For determinism
	sort.Slice(tests, func(i, j int) bool { return *tests[i].ID < *tests[j].ID })

	return v1.NewTestResults(
		v1.PythonPytestFramework,
		tests,
		nil,
	), nil
}

func (p PythonPytestParser) statusOf(testResult PythonPytestTestResult) (*v1.TestStatus, error) {
	var status v1.TestStatus
	switch testResult.Outcome {
	case "failed":
		_, wasxfail := testResult.Keywords["xfail"]

		if wasxfail {
			message := "Unexpectedly passed"
			status = v1.NewFailedTestStatus(&message, nil, nil)
			break
		}

		var longrepr PythonPytestFailedLongrepr
		err := json.Unmarshal(testResult.Longrepr, &longrepr)
		if err != nil {
			fmt.Printf("%s", testResult.Longrepr)
			return nil, errors.Wrap(err, "Unable to parse skipped test longrepr")
		}

		message := longrepr.Reprcrash.Message
		backtrace := []string{fmt.Sprintf("%v:%v", longrepr.Reprcrash.Path, longrepr.Reprcrash.Lineno)}
		status = v1.NewFailedTestStatus(&message, nil, backtrace)
	case "passed":
		status = v1.NewSuccessfulTestStatus()
	case "skipped":
		_, wasxfail := testResult.Keywords["xfail"]

		if wasxfail {
			message := "Expected failure was skipped"
			status = v1.NewSkippedTestStatus(&message)
			break
		}

		var longrepr PythonPytestSkippedLongrepr
		err := json.Unmarshal(testResult.Longrepr, &longrepr)
		if err != nil {
			fmt.Printf("%s", testResult.Longrepr)
			return nil, errors.Wrap(err, "Unable to parse skipped test longrepr")
		}

		skippedMessage := strings.TrimPrefix(longrepr[2].(string), "Skipped: ")
		status = v1.NewSkippedTestStatus(&skippedMessage)
	default:
		return nil, errors.NewInputError("Unexpected test outcome %q", testResult.Outcome)
	}

	return &status, nil
}
