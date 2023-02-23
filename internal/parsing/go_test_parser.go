package parsing

import (
	"bufio"
	"encoding/json"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type GoTestParser struct{}

// https://pkg.go.dev/cmd/test2json
type GoTestTestOutput struct {
	Time    time.Time `json:"Time"`
	Action  *string   `json:"Action"` // bench, cont, fail, output, pass, pause, run, skip
	Package *string   `json:"Package"`
	Test    *string   `json:"Test"`    // optional, depends if the output is about a specific test or package
	Output  *string   `json:"Output"`  // set for output events
	Elapsed *float64  `json:"Elapsed"` // set for pass and fail events
}

func (p GoTestParser) Parse(data io.Reader) (*v1.TestResults, error) {
	testsByPackage := map[string]map[string]v1.Test{}
	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(text, "{") {
			continue
		}

		var testOutput GoTestTestOutput
		if err := json.NewDecoder(strings.NewReader(text)).Decode(&testOutput); err != nil {
			continue
		}

		if testOutput.Action == nil || testOutput.Package == nil {
			return nil, errors.NewInputError("Test results do not look like go test")
		}

		// We don't care about package-level stats/info
		if testOutput.Test == nil {
			continue
		}

		if _, ok := testsByPackage[*testOutput.Package]; !ok {
			testsByPackage[*testOutput.Package] = map[string]v1.Test{}
		}

		testPackage := *testOutput.Package
		existingTest, ok := testsByPackage[testPackage][*testOutput.Test]
		if !ok {
			existingTest = v1.Test{
				Scope: &testPackage,
				Name:  *testOutput.Test,
				Attempt: v1.TestAttempt{
					Meta:   map[string]any{"package": testPackage},
					Status: v1.NewSuccessfulTestStatus(),
				},
			}
		}

		switch *testOutput.Action {
		case "bench":
			// docs indicate this exists, but I can't seem to reproduce
		case "cont":
			// no-op
		case "fail":
			duration := time.Duration(math.Round(*testOutput.Elapsed * float64(time.Second)))
			existingTest.Attempt.Duration = &duration
			existingTest.Attempt.Status = v1.NewFailedTestStatus(nil, nil, nil)
		case "output":
			if testOutput.Output == nil {
				return nil, errors.NewInputError("JSON with action of output is missing output: %v", testOutput)
			}

			if existingTest.Attempt.Stdout == nil {
				existingTest.Attempt.Stdout = testOutput.Output
			} else {
				newStdout := strings.Join([]string{*existingTest.Attempt.Stdout, *testOutput.Output}, "")
				existingTest.Attempt.Stdout = &newStdout
			}
		case "pass":
			duration := time.Duration(math.Round(*testOutput.Elapsed * float64(time.Second)))
			existingTest.Attempt.Duration = &duration
			existingTest.Attempt.Status = v1.NewSuccessfulTestStatus()
		case "pause":
			// no-op
		case "run":
			// no-op
		case "skip":
			duration := time.Duration(math.Round(*testOutput.Elapsed * float64(time.Second)))
			existingTest.Attempt.Duration = &duration
			existingTest.Attempt.Status = v1.NewSkippedTestStatus(nil)
		default:
			return nil, errors.NewInputError("Unexpected test action: %v", *testOutput.Action)
		}

		testsByPackage[*testOutput.Package][*testOutput.Test] = existingTest
	}

	tests := make([]v1.Test, 0)
	for _, testsByName := range testsByPackage {
		for _, test := range testsByName {
			tests = append(tests, test)
		}
	}

	if len(tests) == 0 {
		return nil, errors.NewInputError("Did not see any tests, so we cannot be sure it is go test output")
	}

	// For determinism
	sort.Slice(tests, func(i, j int) bool {
		if tests[i].Name == tests[j].Name {
			return tests[i].Attempt.Meta["package"].(string) < tests[j].Attempt.Meta["package"].(string)
		}

		return tests[i].Name < tests[j].Name
	})

	return v1.NewTestResults(
		v1.GoTestFramework,
		tests,
		nil,
	), nil
}
