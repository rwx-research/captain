package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type RubyMinitestSubstitution struct{}

func (s RubyMinitestSubstitution) Example() string {
	return "bin/rails test {{ tests }}"
}

func (s RubyMinitestSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying minitest requires a template with the 'tests' keyword " +
				"or a template with the 'file' and 'name' keywords; " +
				"no keywords were found",
		)
	}

	if len(keywords) > 2 {
		return errors.NewInputError(
			"Retrying minitest requires a template with the 'tests' keyword "+
				"or a template with the 'file' and 'name' keywords; "+
				"these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if len(keywords) == 1 {
		keyword := keywords[0]

		if keyword != "tests" {
			return errors.NewInputError(
				"Retrying minitest requires a template with the 'tests' keyword "+
					"or a template with the 'file' and 'name' keywords; "+
					"'%v' was found",
				keyword,
			)
		}

		return nil
	}

	if (keywords[0] == "file" && keywords[1] == "name") ||
		(keywords[0] == "name" && keywords[1] == "file") {
		return nil
	}

	return errors.NewInputError(
		"Retrying minitest requires a template with the 'tests' keyword "+
			"or a template with the 'file' and 'name' keywords; "+
			"'%v' and '%v were found",
		keywords[0],
		keywords[1],
	)
}

func (s RubyMinitestSubstitution) SubstitutionsFor(
	compiledTemplate CompiledTemplate,
	testResults v1.TestResults,
) []map[string]string {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 1 {
		return s.allTestsSubstitutions(testResults)
	}

	return s.singleTestSubstitutions(testResults)
}

func (s RubyMinitestSubstitution) allTestsSubstitutions(testResults v1.TestResults) []map[string]string {
	tests := make([]string, 0)

	for _, test := range testResults.Tests {
		if test.Attempt.Status.ImpliesFailure() {
			tests = append(
				tests,
				fmt.Sprintf("'%v:%v'", ShellEscape(test.Location.File), *test.Location.Line),
			)
		}
	}

	return []map[string]string{{"tests": strings.Join(tests, " ")}}
}

func (s RubyMinitestSubstitution) singleTestSubstitutions(testResults v1.TestResults) []map[string]string {
	testsSeenByFile := map[string]map[string]struct{}{}

	for _, test := range testResults.Tests {
		if !test.Attempt.Status.ImpliesFailure() {
			continue
		}

		file := ShellEscape(test.Location.File)
		if _, ok := testsSeenByFile[file]; !ok {
			testsSeenByFile[file] = map[string]struct{}{}
		}

		methodName := ShellEscape(test.Lineage[len(test.Lineage)-1])
		if _, ok := testsSeenByFile[file][methodName]; ok {
			continue
		}

		testsSeenByFile[file][methodName] = struct{}{}
	}

	substitutions := make([]map[string]string, 0)
	for file, testsSeen := range testsSeenByFile {
		for methodName := range testsSeen {
			substitutions = append(substitutions, map[string]string{
				"file": file,
				"name": methodName,
			})
		}
	}

	return substitutions
}
