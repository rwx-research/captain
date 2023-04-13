package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type RubyCucumberSubstitution struct{}

func (s RubyCucumberSubstitution) Example() string {
	return "bundle exec cucumber {{ examples }}"
}

func (s RubyCucumberSubstitution) ValidateTemplate(compiledTemplate templating.CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying Cucumber requires a template with the 'examples' keyword; no keywords were found",
		)
	}

	if len(keywords) > 1 {
		return errors.NewInputError(
			"Retrying Cucumber requires a template with only the 'examples' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if keywords[0] != "examples" {
		return errors.NewInputError(
			"Retrying Cucumber requires a template with only the 'examples' keyword; '%v' was found instead",
			keywords[0],
		)
	}

	return nil
}

func (s RubyCucumberSubstitution) SubstitutionsFor(
	_ templating.CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	examples := make([]string, 0)
	examplesSeen := map[string]struct{}{}

	for _, test := range testResults.Tests {
		if test.Attempt.Status.ImpliesFailure() && filter(test) {
			example := templating.ShellEscape(test.Attempt.Meta["elementStart"].(string))
			if _, ok := examplesSeen[example]; ok {
				continue
			}

			examples = append(examples, fmt.Sprintf("'%v'", example))
			examplesSeen[example] = struct{}{}
		}
	}

	if len(examples) > 0 {
		return []map[string]string{{"examples": strings.Join(examples, " ")}}, nil
	}

	return []map[string]string{}, nil
}
