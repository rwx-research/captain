package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type PythonPytestSubstitution struct{}

func (s PythonPytestSubstitution) Example() string {
	return "pytest {{ tests }}"
}

func (s PythonPytestSubstitution) ValidateTemplate(compiledTemplate templating.CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError("Retrying pytest requires a template with the 'tests' keyword; no keywords were found")
	}

	if len(keywords) > 1 {
		return errors.NewInputError(
			"Retrying pytest requires a template with only the 'tests' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if keywords[0] != "tests" {
		return errors.NewInputError(
			"Retrying pytest requires a template with only the 'tests' keyword; '%v' was found instead",
			keywords[0],
		)
	}

	return nil
}

func (s PythonPytestSubstitution) SubstitutionsFor(
	_ templating.CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	testIdentifiers := make([]string, 0)

	for _, test := range testResults.Tests {
		if test.Attempt.Status.ImpliesFailure() && filter(test) {
			testIdentifiers = append(
				testIdentifiers,
				fmt.Sprintf("'%v'", templating.ShellEscape(*test.ID)),
			)
		}
	}

	if len(testIdentifiers) > 0 {
		return []map[string]string{{"tests": strings.Join(testIdentifiers, " ")}}, nil
	}

	return []map[string]string{}, nil
}
