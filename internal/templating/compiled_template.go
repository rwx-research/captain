package templating

import (
	"regexp"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
)

var (
	placeholderRegexp = regexp.MustCompile(`({{\s?\w+\s?}})`)
	keywordRegexp     = regexp.MustCompile(`^{{\s?(\w+)\s?}}$`)
)

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

func (ct CompiledTemplate) Substitute(substitutionLookup map[string]string) string {
	substituted := ct.Template
	for placeholder, keyword := range ct.PlaceholderToKeyword {
		substituted = strings.Replace(substituted, placeholder, substitutionLookup[keyword], 1)
	}
	return substituted
}
