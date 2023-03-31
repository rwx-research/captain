package targetedretries

import (
	"encoding/json"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JSONSubstitution struct {
	FileSystem fs.FileSystem
}

func (s JSONSubstitution) Example() string {
	return "bin/your-script {{ jsonFilePath }}"
}

func (s JSONSubstitution) ValidateTemplate(compiledTemplate CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying with JSON requires a template with the 'jsonFilePath' keyword; no keywords were found",
		)
	}

	if len(keywords) > 1 {
		return errors.NewInputError(
			"Retrying with JSON requires a template with only the 'jsonFilePath' keyword; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	if keywords[0] != "jsonFilePath" {
		return errors.NewInputError(
			"Retrying with JSON requires a template with only the 'jsonFilePath' keyword; '%v' was found instead",
			keywords[0],
		)
	}

	return nil
}

func (s JSONSubstitution) SubstitutionsFor(
	_ CompiledTemplate,
	testResults v1.TestResults,
	filter func(test v1.Test) bool,
) ([]map[string]string, error) {
	testsToRetry := make([]v1.Test, 0)

	for _, test := range testResults.Tests {
		if test.Attempt.Status.ImpliesFailure() && filter(test) {
			testsToRetry = append(testsToRetry, test)
		}
	}

	file, err := s.FileSystem.CreateTemp("", "test-results-to-retry")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer file.Close()

	testResultsToRetry := v1.NewTestResults(testResults.Framework, testsToRetry, nil)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(testResultsToRetry); err != nil {
		return nil, errors.WithStack(err)
	}

	return []map[string]string{{"jsonFilePath": file.Name()}}, nil
}

func (s JSONSubstitution) CleanUp(allSubstitutions []map[string]string) error {
	if len(allSubstitutions) == 0 {
		return nil
	}

	if len(allSubstitutions) > 1 {
		return errors.NewInternalError(
			"Expected JSONSubstitution to create only one substitution, but got %v",
			len(allSubstitutions),
		)
	}

	substitutions := allSubstitutions[0]
	filePath, ok := substitutions["jsonFilePath"]
	if !ok {
		return errors.NewInternalError(
			"Expected JSONSubstitution to expose the file path in the substitution, but it did not",
		)
	}

	err := s.FileSystem.Remove(filePath)
	return errors.Wrapf(err, "Unabled to clean up %q", filePath)
}
