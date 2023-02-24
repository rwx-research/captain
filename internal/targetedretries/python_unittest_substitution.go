package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type PythonUnitTestSubstitution struct{}

func (s PythonUnitTestSubstitution) Example() string {
	return "python -m xmlrunner {{ tests }}"
}

func (s PythonUnitTestSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError("Retrying unittest requires a template with the 'tests' keyword; no keywords were found")
	}

	if len(keywords) > 1 {
		return errors.NewInputError(
			"Retrying unittest requires a template with only the 'tests' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if keywords[0] != "tests" {
		return errors.NewInputError(
			"Retrying unittest requires a template with only the 'tests' keyword; '%v' was found instead",
			keywords[0],
		)
	}

	return nil
}

func (s PythonUnitTestSubstitution) SubstitutionsFor(
	compiledTemplate CompiledTemplate,
	testResults v1.TestResults,
) []map[string]string {
	testIdentifiers := make([]string, 0)

	for _, test := range testResults.Tests {
		if test.Attempt.Status.ImpliesFailure() {
			testIdentifiers = append(
				testIdentifiers,
				fmt.Sprintf("'%v'", ShellEscape(test.Name)),
			)
		}
	}

	return []map[string]string{{"tests": strings.Join(testIdentifiers, " ")}}
}
