package targetedretries

import (
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type PHPUnitSubstitution struct{}

func (s PHPUnitSubstitution) Example() string {
	return "vendor/bin/phpunit --filter '{{ filter }}' '{{ file }}'"
}

func (s PHPUnitSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying Mocha requires a template with the 'filter' and 'file' keywords; no keywords were found",
		)
	}

	if len(keywords) != 2 {
		return errors.NewInputError(
			"Retrying Mocha requires a template with the 'filter' and 'file' keywords; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if (keywords[0] == "filter" && keywords[1] == "file") ||
		(keywords[0] == "file" && keywords[1] == "filter") {
		return nil
	}

	return errors.NewInputError(
		"Retrying Mocha requires a template with the 'filter' and 'file' keywords; '%v' and '%v' were found instead",
		keywords[0],
		keywords[1],
	)
}

func (s PHPUnitSubstitution) SubstitutionsFor(
	compiledTemplate CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
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

		methodName := test.Attempt.Meta["name"].(string)
		if _, ok := testsSeenByFile[file][methodName]; ok {
			continue
		}

		testsSeenByFile[file][methodName] = struct{}{}
	}

	substitutions := make([]map[string]string, 0)
	for file, testsSeen := range testsSeenByFile {
		for methodName := range testsSeen {
			substitutions = append(substitutions, map[string]string{
				"file":   file,
				"filter": methodName,
			})
		}
	}

	return substitutions, nil
}
