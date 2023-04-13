package partition

import (
	"github.com/rwx-research/captain-cli/internal/templating"
)

type Substitution interface {
	Example() string
	ValidateTemplate(compiledTemplate templating.CompiledTemplate) error
	SubstitutionLookupFor(
		_ templating.CompiledTemplate,
		testFilePaths []string,
	) (map[string]string, error)
}
