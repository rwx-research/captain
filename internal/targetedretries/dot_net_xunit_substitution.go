package targetedretries

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type DotNetxUnitSubstitution struct{}

func (s DotNetxUnitSubstitution) Example() string {
	return "dotnet test --filter '{{ filter }}'"
}

func (s DotNetxUnitSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError("Retrying xUnit requires a template with the 'filter' keyword; no keywords were found")
	}

	if len(keywords) > 1 {
		return errors.NewInputError(
			"Retrying xUnit requires a template with only the 'filter' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if keywords[0] != "filter" {
		return errors.NewInputError(
			"Retrying xUnit requires a template with only the 'filter' keyword; '%v' was found instead",
			keywords[0],
		)
	}

	return nil
}

func (s DotNetxUnitSubstitution) SubstitutionsFor(
	_ CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	testsSeen := map[string]struct{}{}
	tests := make([]string, 0)

	for _, test := range testResults.Tests {
		if !test.Attempt.Status.ImpliesFailure() {
			continue
		}
		if !filter(test) {
			continue
		}

		testType := test.Attempt.Meta["type"].(*string)
		testMethod := test.Attempt.Meta["method"].(*string)
		formattedTest := fmt.Sprintf("FullyQualifiedName=%v.%v", *testType, *testMethod)
		if _, ok := testsSeen[formattedTest]; ok {
			continue
		}

		tests = append(tests, formattedTest)
		testsSeen[formattedTest] = struct{}{}
	}

	return []map[string]string{{"filter": strings.Join(tests, " | ")}}, nil
}
