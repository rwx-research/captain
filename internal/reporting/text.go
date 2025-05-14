package reporting

import (
	"fmt"
	"unicode"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

func WriteTextSummary(file fs.File, testResults v1.TestResults, _ Configuration) error {
	statuses := make(map[v1.TestStatusKind][]string)
	totalTests := testResults.Summary.Tests

	for _, test := range testResults.Tests {
		if test.Attempt.Status.Kind == v1.TestStatusSuccessful {
			continue
		}

		tests, ok := statuses[test.Attempt.Status.Kind]
		if !ok {
			tests = make([]string, 0)
		}

		tests = append(tests, test.Name)
		statuses[test.Attempt.Status.Kind] = tests
	}

	pluralizeTests := "tests"
	if totalTests == 1 {
		pluralizeTests = "test"
	}

	_, err := file.Write([]byte(fmt.Sprintf("Captain detected a total of %d %s.\n", totalTests, pluralizeTests)))
	if err != nil {
		return errors.WithStack(err)
	}

	for status, tests := range statuses {
		statusName := []rune(status)
		statusName[0] = unicode.ToUpper(statusName[0])

		_, err := file.Write([]byte(fmt.Sprintf("\n%s (%d):\n", string(statusName), len(tests))))
		if err != nil {
			return errors.WithStack(err)
		}

		for _, testName := range tests {
			_, err := file.Write([]byte(fmt.Sprintf("- %s\n", testName)))
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
