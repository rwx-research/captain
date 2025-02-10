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
	return "bundle exec cucumber {{ scenarios }}"
}

func (s RubyCucumberSubstitution) ValidateTemplate(compiledTemplate templating.CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying Cucumber requires a template with the 'scenarios' keyword; no keywords were found",
		)
	}

	if len(keywords) > 1 {
		return errors.NewInputError(
			"Retrying Cucumber requires a template with only the 'scenarios' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if keywords[0] != "scenarios" {
		return errors.NewInputError(
			"Retrying Cucumber requires a template with only the 'scenarios' keyword; '%v' was found instead",
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
	scenarios := make([]string, 0)
	scenariosSeen := map[string]struct{}{}

	for _, test := range testResults.Tests {
		if filter(test) {
			scenarioLocation := test.Location.File
			if test.Location.Line != nil {
				scenarioLocation = fmt.Sprintf("%s:%v", test.Location.File, *test.Location.Line)
			}
			example := templating.ShellEscape(scenarioLocation)
			if _, ok := scenariosSeen[example]; ok {
				continue
			}

			scenarios = append(scenarios, fmt.Sprintf("'%v'", example))
			scenariosSeen[example] = struct{}{}
		}
	}

	if len(scenarios) > 0 {
		return []map[string]string{{"scenarios": strings.Join(scenarios, " ")}}, nil
	}

	return []map[string]string{}, nil
}
