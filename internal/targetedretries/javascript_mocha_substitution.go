package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JavaScriptMochaSubstitution struct{}

func (s JavaScriptMochaSubstitution) Example() string {
	return "npx mocha '{{ file }}' --grep '{{ grep }}'"
}

func (s JavaScriptMochaSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying Mocha requires a template with the 'file' and 'grep' keywords; no keywords were found",
		)
	}

	if len(keywords) != 2 {
		return errors.NewInputError(
			"Retrying Mocha requires a template with the 'file' and 'grep' keywords; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if (keywords[0] == "file" && keywords[1] == "grep") ||
		(keywords[0] == "grep" && keywords[1] == "file") {
		return nil
	}

	return errors.NewInputError(
		"Retrying Mocha requires a template with the 'file' and 'grep' keywords; '%v' and '%v' were found instead",
		keywords[0],
		keywords[1],
	)
}

func (s JavaScriptMochaSubstitution) SubstitutionsFor(
	compiledTemplate CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) []map[string]string {
	testsByFile := map[string][]string{}
	testsSeenByFile := map[string]map[string]struct{}{}

	for _, test := range testResults.Tests {
		if !test.Attempt.Status.ImpliesFailure() {
			continue
		}
		if !filter(test) {
			continue
		}

		file := ShellEscape(test.Location.File)
		if _, ok := testsSeenByFile[file]; !ok {
			testsSeenByFile[file] = map[string]struct{}{}
		}
		if _, ok := testsByFile[file]; !ok {
			testsByFile[file] = make([]string, 0)
		}

		formattedName := ShellEscape(RegexpEscape(test.Name))
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
			"file": file,
			"grep": fmt.Sprintf("^%v$", strings.Join(tests, "|")),
		}
		i++
	}

	return substitutions
}
