package targetedretries

import (
	"regexp"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type Substitution interface {
	Example() string
	ValidateTemplate(compiledTemplate CompiledTemplate) error
	SubstitutionsFor(
		_ CompiledTemplate,
		testResults v1.TestResults,
		filter func(v1.Test) bool,
	) ([]map[string]string, error)
}

var (
	placeholderRegexp = regexp.MustCompile(`({{\s?\w+\s?}})`)
	keywordRegexp     = regexp.MustCompile(`^{{\s?(\w+)\s?}}$`)
)

var SubstitutionsByFramework = map[v1.Framework]Substitution{
	v1.DotNetxUnitFramework:          new(DotNetxUnitSubstitution),
	v1.ElixirExUnitFramework:         new(ElixirExUnitSubstitution),
	v1.GoGinkgoFramework:             new(GoGinkgoSubstitution),
	v1.GoTestFramework:               new(GoTestSubstitution),
	v1.JavaScriptCypressFramework:    new(JavaScriptCypressSubstitution),
	v1.JavaScriptJestFramework:       new(JavaScriptJestSubstitution),
	v1.JavaScriptMochaFramework:      new(JavaScriptMochaSubstitution),
	v1.JavaScriptPlaywrightFramework: new(JavaScriptPlaywrightSubstitution),
	v1.PHPUnitFramework:              new(PHPUnitSubstitution),
	v1.PythonPytestFramework:         new(PythonPytestSubstitution),
	v1.PythonUnitTestFramework:       new(PythonUnitTestSubstitution),
	v1.RubyCucumberFramework:         new(RubyCucumberSubstitution),
	v1.RubyMinitestFramework:         new(RubyMinitestSubstitution),
	v1.RubyRSpecFramework:            new(RubyRSpecSubstitution),
}

type CompiledTemplate struct {
	Template             string
	PlaceholderToKeyword map[string]string
}

func CompileTemplate(template string) (CompiledTemplate, error) {
	placeholders := placeholderRegexp.FindAllString(template, -1)
	if len(placeholders) == 0 {
		return CompiledTemplate{Template: template, PlaceholderToKeyword: map[string]string{}}, nil
	}

	keywordsSubstituted := make(map[string]struct{}, len(placeholders))
	placeholderToKeyword := make(map[string]string, len(placeholders))
	for _, placeholder := range placeholders {
		submatches := keywordRegexp.FindStringSubmatch(placeholder)
		if len(submatches) != 2 {
			return CompiledTemplate{}, errors.NewInputError(
				"template included a malformed placeholder '%v'",
				placeholder,
			)
		}

		keyword := submatches[1]
		if _, ok := keywordsSubstituted[keyword]; ok {
			return CompiledTemplate{}, errors.NewInputError(
				"template requested duplicate substitution of placeholder '%v'",
				keyword,
			)
		}
		keywordsSubstituted[keyword] = struct{}{}
		placeholderToKeyword[placeholder] = keyword
	}

	return CompiledTemplate{Template: template, PlaceholderToKeyword: placeholderToKeyword}, nil
}

func (ct CompiledTemplate) Keywords() []string {
	keywords := make([]string, len(ct.PlaceholderToKeyword))

	i := 0
	for _, keyword := range ct.PlaceholderToKeyword {
		keywords[i] = keyword
		i++
	}

	return keywords
}

func (ct CompiledTemplate) Substitute(substitutions map[string]string) string {
	substituted := ct.Template
	for placeholder, keyword := range ct.PlaceholderToKeyword {
		substituted = strings.Replace(substituted, placeholder, substitutions[keyword], 1)
	}
	return substituted
}
