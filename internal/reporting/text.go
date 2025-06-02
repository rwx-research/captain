package reporting

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

var detailedTestStatusKinds = []v1.TestStatusKind{
	v1.TestStatusFailed,
	v1.TestStatusTimedOut,
	v1.TestStatusCanceled,
}

var orderedTestStatusKinds = []v1.TestStatusKind{
	v1.TestStatusSuccessful,
	v1.TestStatusFailed,
	v1.TestStatusTimedOut,
	v1.TestStatusCanceled,
	v1.TestStatusQuarantined,
	v1.TestStatusSkipped,
	v1.TestStatusPended,
	v1.TestStatusTodo,
}

var titleCaser = cases.Title(language.AmericanEnglish)

func WriteTextSummary(w io.Writer, testResults v1.TestResults, _ Configuration) error {
	statuses := summarizeTestsByStatus(testResults)
	totalTests := testResults.Summary.Tests

	for _, kind := range detailedTestStatusKinds {
		tests, ok := statuses[kind]
		if !ok {
			continue
		}

		statusName := titleCaser.String(testStatusKindToString(kind))
		_, err := fmt.Fprintf(w, "\n%s (%d):\n", statusName, len(statuses[kind]))
		if err != nil {
			return errors.WithStack(err)
		}

		for _, testName := range tests {
			_, err := fmt.Fprintf(w, "- %s\n", testName)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	totalCountParts := make([]string, 0, len(orderedTestStatusKinds))
	if testResults.Summary.Successful > 0 {
		totalCountParts = append(totalCountParts, fmt.Sprintf("%d %s", testResults.Summary.Successful, "successful"))
	}

	for _, kind := range orderedTestStatusKinds {
		tests, ok := statuses[kind]
		if !ok || len(tests) == 0 {
			continue
		}

		totalCountParts = append(totalCountParts, fmt.Sprintf("%d %s", len(tests), testStatusKindToString(kind)))
	}

	pluralizeTests := "tests"
	if totalTests == 1 {
		pluralizeTests = "test"
	}

	_, err := fmt.Fprintf(w, "\n%d total %s", totalTests, pluralizeTests)
	if err != nil {
		return errors.WithStack(err)
	}

	if len(totalCountParts) > 0 {
		_, err = fmt.Fprintf(w, ": %s\n", strings.Join(totalCountParts, ", "))
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		_, err = fmt.Fprint(w, "\n")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func summarizeTestsByStatus(testResults v1.TestResults) map[v1.TestStatusKind][]string {
	statuses := make(map[v1.TestStatusKind][]string)

	for _, test := range testResults.Tests {
		if test.Attempt.Status.Kind == v1.TestStatusSuccessful {
			continue
		}

		tests, ok := statuses[test.Attempt.Status.Kind]
		if !ok {
			tests = []string{test.Name}
		} else {
			tests = append(tests, test.Name)
		}

		statuses[test.Attempt.Status.Kind] = tests
	}

	return statuses
}

func testStatusKindToString(kind v1.TestStatusKind) string {
	switch kind { //nolint:exhaustive
	case v1.TestStatusTimedOut:
		return "timed out"
	default:
		return string(kind)
	}
}
