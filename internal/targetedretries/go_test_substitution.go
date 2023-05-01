package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type GoTestSubstitution struct{}

func (s GoTestSubstitution) Example() string {
	return "go test '{{ package }}' -run '{{ run }}'"
}

func (s GoTestSubstitution) SuffixExample() string {
	return "'{{ package }}' -run '{{ run }}'"
}

func (s GoTestSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying go test requires a template with the 'package' and 'run' keywords; no keywords were found",
		)
	}

	if len(keywords) != 2 {
		return errors.NewInputError(
			"Retrying go test requires a template with the 'package' and 'run' keywords; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if (keywords[0] == "package" && keywords[1] == "run") || (keywords[0] == "run" && keywords[1] == "package") {
		return nil
	}

	return errors.NewInputError(
		"Retrying go test requires a template with the 'package' and 'run' keywords; '%v' and '%v' were found instead",
		keywords[0],
		keywords[1],
	)
}

func (s GoTestSubstitution) SubstitutionsFor(
	_ CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	testsByPackage := map[string][]string{}
	testsSeenByPackage := map[string]map[string]struct{}{}

	for _, test := range testResults.Tests {
		if !test.Attempt.Status.ImpliesFailure() {
			continue
		}
		if !filter(test) {
			continue
		}

		testPackage := ShellEscape(test.Attempt.Meta["package"].(string))
		testName := test.Name
		if strings.Contains(testName, "/") {
			// Table tests cannot be run individually, so run them all
			testName = strings.Split(testName, "/")[0]
		}

		if _, ok := testsSeenByPackage[testPackage]; !ok {
			testsSeenByPackage[testPackage] = map[string]struct{}{}
		}
		if _, ok := testsByPackage[testPackage]; !ok {
			testsByPackage[testPackage] = make([]string, 0)
		}

		formattedTest := RegexpEscape(testName)
		if _, ok := testsSeenByPackage[testPackage][formattedTest]; ok {
			continue
		}

		testsByPackage[testPackage] = append(testsByPackage[testPackage], formattedTest)
		testsSeenByPackage[testPackage][formattedTest] = struct{}{}
	}

	substitutions := make([]map[string]string, len(testsByPackage))
	i := 0
	for testPackage, tests := range testsByPackage {
		substitutions[i] = map[string]string{
			"package": testPackage,
			"run":     fmt.Sprintf("^%v$", strings.Join(tests, "|")),
		}
		i++
	}

	return substitutions, nil
}
