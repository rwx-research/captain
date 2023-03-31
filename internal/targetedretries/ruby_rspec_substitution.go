package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type RubyRSpecSubstitution struct{}

func (s RubyRSpecSubstitution) Example() string {
	return "bundle exec rspec {{ tests }}"
}

func (s RubyRSpecSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError("Retrying RSpec requires a template with the 'tests' keyword; no keywords were found")
	}

	if len(keywords) > 1 {
		return errors.NewInputError(
			"Retrying RSpec requires a template with only the 'tests' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if keywords[0] != "tests" {
		return errors.NewInputError(
			"Retrying RSpec requires a template with only the 'tests' keyword; '%v' was found instead",
			keywords[0],
		)
	}

	return nil
}

func (s RubyRSpecSubstitution) SubstitutionsFor(
	_ CompiledTemplate,
	testResults v1.TestResults,
	filter func(test v1.Test) bool,
) ([]map[string]string, error) {
	testIdentifiers := make([]string, 0)

	for _, test := range testResults.Tests {
		if test.Attempt.Status.ImpliesFailure() && filter(test) {
			testIdentifiers = append(
				testIdentifiers,
				fmt.Sprintf("'%v'", ShellEscape(*test.ID)),
			)
		}
	}

	return []map[string]string{{"tests": strings.Join(testIdentifiers, " ")}}, nil
}
