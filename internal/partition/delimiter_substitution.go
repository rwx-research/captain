package partition

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/templating"
)

type DelimiterSubstitution struct {
	Delimiter string
}

func (s DelimiterSubstitution) Example() string {
	return "your-test-command {{ testFiles }}"
}

func (s DelimiterSubstitution) ValidateTemplate(compiledTemplate templating.CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()
	message := "Partitioning with a delimiter requires a template with only the 'testFiles' keyword"

	if len(keywords) == 0 {
		return errors.NewInputError("%v; no keywords were found", message)
	}

	if len(keywords) > 1 {
		return errors.NewInputError("%v; these were found: %v", message, strings.Join(keywords, ", "))
	}

	if keywords[0] != "testFiles" {
		return errors.NewInputError("%v; '%v' was found instead", message, keywords[0])
	}

	return nil
}

func (s DelimiterSubstitution) SubstitutionLookupFor(
	_ templating.CompiledTemplate,
	testFilePaths []string,
) (map[string]string, error) {
	escapedTestFilePaths := make([]string, 0)

	for _, testFilePath := range testFilePaths {
		escapedTestFilePaths = append(escapedTestFilePaths, fmt.Sprintf("'%v'", templating.ShellEscape(testFilePath)))
	}

	return map[string]string{"testFiles": strings.Join(escapedTestFilePaths, s.Delimiter)}, nil
}
