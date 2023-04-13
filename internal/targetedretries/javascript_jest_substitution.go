package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JavaScriptJestSubstitution struct{}

func (s JavaScriptJestSubstitution) Example() string {
	return "npx jest --testPathPattern '{{ testPathPattern }}' --testNamePattern '{{ testNamePattern }}'"
}

func (s JavaScriptJestSubstitution) ValidateTemplate(compiledTemplate templating.CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying Jest requires a template with the 'testPathPattern' and 'testNamePattern' keywords; " +
				"no keywords were found",
		)
	}

	if len(keywords) != 2 {
		return errors.NewInputError(
			"Retrying Jest requires a template with the 'testPathPattern' and 'testNamePattern' keywords; "+
				"these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if (keywords[0] == "testPathPattern" && keywords[1] == "testNamePattern") ||
		(keywords[0] == "testNamePattern" && keywords[1] == "testPathPattern") {
		return nil
	}

	return errors.NewInputError(
		"Retrying Jest requires a template with the 'testPathPattern' and 'testNamePattern' keywords; "+
			"'%v' and '%v' were found instead",
		keywords[0],
		keywords[1],
	)
}

func (s JavaScriptJestSubstitution) SubstitutionsFor(
	_ templating.CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	testsByFile := map[string][]string{}
	testsSeenByFile := map[string]map[string]struct{}{}

	for _, test := range testResults.Tests {
		if !test.Attempt.Status.ImpliesFailure() {
			continue
		}
		if !filter(test) {
			continue
		}

		file := templating.ShellEscape(test.Location.File)
		if _, ok := testsSeenByFile[file]; !ok {
			testsSeenByFile[file] = map[string]struct{}{}
		}
		if _, ok := testsByFile[file]; !ok {
			testsByFile[file] = make([]string, 0)
		}

		formattedName := templating.ShellEscape(templating.RegexpEscape(strings.Join(test.Lineage, " ")))
		if _, ok := testsSeenByFile[file][formattedName]; ok {
			continue
		}

		testsByFile[file] = append(testsByFile[file], formattedName)
		testsSeenByFile[file][formattedName] = struct{}{}
	}

	substitutions := make([]map[string]string, len(testsByFile))
	i := 0
	for file, tests := range testsByFile {
		substitutions[i] = map[string]string{
			"testPathPattern": file,
			"testNamePattern": fmt.Sprintf("^%v$", strings.Join(tests, "|")),
		}
		i++
	}

	return substitutions, nil
}
