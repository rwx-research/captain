package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JavaScriptCypressSubstitution struct{}

func (s JavaScriptCypressSubstitution) Example() string {
	return "npx cypress run --spec '{{ spec }}' --env {{ grep }}"
}

func (s JavaScriptCypressSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying Cypress requires a template with the 'spec' and optionally the 'grep' keyword; no keywords were found",
		)
	}

	if len(keywords) > 2 {
		return errors.NewInputError(
			"Retrying Cypress requires a template with the 'spec' and optionally the 'grep' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if len(keywords) == 1 {
		keyword := keywords[0]

		if keyword != "spec" {
			return errors.NewInputError(
				"Retrying Cypress requires a template with the 'spec' and optionally the 'grep' keyword; only '%v' was found",
				keyword,
			)
		}

		return nil
	}

	firstKeyword := keywords[0]
	secondKeyword := keywords[1]
	if (firstKeyword == "spec" && secondKeyword == "grep") || (firstKeyword == "grep" && secondKeyword == "spec") {
		return nil
	}

	return errors.NewInputError(
		"Retrying Cypress requires a template with the 'spec' and optionally the 'grep' keyword; these were found: %v",
		strings.Join(keywords, ", "),
	)
}

func (s JavaScriptCypressSubstitution) SubstitutionsFor(
	compiledTemplate CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	testsBySpec := map[string][]string{}
	testsSeenBySpec := map[string]map[string]struct{}{}

	for _, test := range testResults.Tests {
		if !test.Attempt.Status.ImpliesFailure() {
			continue
		}
		if !filter(test) {
			continue
		}

		spec := ShellEscape(test.Location.File)
		if _, ok := testsSeenBySpec[spec]; !ok {
			testsSeenBySpec[spec] = map[string]struct{}{}
		}
		if _, ok := testsBySpec[spec]; !ok {
			testsBySpec[spec] = make([]string, 0)
		}

		name := ShellEscape(test.Name)
		if _, ok := testsSeenBySpec[spec][name]; ok {
			continue
		}

		testsBySpec[spec] = append(testsBySpec[spec], name)
		testsSeenBySpec[spec][name] = struct{}{}
	}

	substitutions := make([]map[string]string, len(testsBySpec))
	i := 0
	for spec, tests := range testsBySpec {
		substitutions[i] = map[string]string{
			"spec": spec,
			"grep": fmt.Sprintf("grep='%v'", strings.Join(tests, "; ")),
		}
		i++
	}

	return substitutions, nil
}
