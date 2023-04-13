package targetedretries

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type ElixirExUnitSubstitution struct{}

func (s ElixirExUnitSubstitution) Example() string {
	return "mix test {{ tests }}"
}

func (s ElixirExUnitSubstitution) ValidateTemplate(compiledTemplate templating.CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError("Retrying ExUnit requires a template with the 'tests' keyword; no keywords were found")
	}

	if len(keywords) > 1 {
		return errors.NewInputError(
			"Retrying ExUnit requires a template with only the 'tests' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if keywords[0] != "tests" {
		return errors.NewInputError(
			"Retrying ExUnit requires a template with only the 'tests' keyword; '%v' was found instead",
			keywords[0],
		)
	}

	return nil
}

func (s ElixirExUnitSubstitution) SubstitutionsFor(
	_ templating.CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	testLinesByFile := map[string][]string{}

	for _, test := range testResults.Tests {
		if !test.Attempt.Status.ImpliesFailure() {
			continue
		}
		if !filter(test) {
			continue
		}

		file := ShellEscape(test.Location.File)
		if _, ok := testLinesByFile[file]; !ok {
			testLinesByFile[file] = make([]string, 0)
		}

		testLinesByFile[file] = append(testLinesByFile[file], strconv.Itoa(*test.Location.Line))
	}

	substitutions := make([]map[string]string, len(testLinesByFile))
	i := 0
	for file, testLines := range testLinesByFile {
		substitutions[i] = map[string]string{
			"tests": fmt.Sprintf("'%v:%v'", file, strings.Join(testLines, ":")),
		}
		i++
	}

	return substitutions, nil
}
