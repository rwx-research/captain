package targetedretries

import (
	"regexp"
	"sort"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type PestSubstitution struct{}

func (s PestSubstitution) Example() string {
	return "vendor/bin/pest --filter '{{ filter }}' '{{ file }}'"
}

func (s PestSubstitution) ValidateTemplate(compiledTemplate templating.CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()
	if len(keywords) == 2 && ((keywords[0] == "filter" && keywords[1] == "file") ||
		(keywords[0] == "file" && keywords[1] == "filter")) {
		return nil
	}
	return errors.NewInputError("Retrying Pest requires a template with only the 'filter' and 'file' keywords")
}

func (s PestSubstitution) SubstitutionsFor(
	_ templating.CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	namesByFile := make(map[string]map[string]bool)
	for _, test := range testResults.Tests {
		if !filter(test) {
			continue
		}
		if test.Location == nil || !strings.HasSuffix(test.Location.File, ".php") {
			return nil, errors.NewInputError(
				"Pest retries require a physical PHP filename for %q; run native PHPUnit tests as a separate PHPUnit suite",
				test.Name,
			)
		}
		file := test.Location.File
		if namesByFile[file] == nil {
			namesByFile[file] = make(map[string]bool)
		}
		class := test.Attempt.Meta["class"].(string)
		name := test.Attempt.Meta["name"].(string)
		namesByFile[file][class+"::"+name] = true
		// Pest 4 trims the filter target; Pest 3 preserves trailing whitespace.
		namesByFile[file][strings.TrimSpace(class+"::"+name)] = true
		// Pest 3 omits dataset labels from filter targets, so retry the enclosing test there.
		// Pest 4 includes them and matches only the full failed dataset name above.
		if datasetName, ok := test.Attempt.Meta["pestDatasetName"].(string); ok {
			namesByFile[file][class+"::"+datasetName] = true
		}
	}
	files := make([]string, 0, len(namesByFile))
	for file := range namesByFile {
		files = append(files, file)
	}
	sort.Strings(files)
	substitutions := make([]map[string]string, 0, len(files))
	for _, file := range files {
		names := make([]string, 0, len(namesByFile[file]))
		for name := range namesByFile[file] {
			names = append(names, strings.ReplaceAll(regexp.QuoteMeta(name), "/", `\/`))
		}
		sort.Strings(names)
		pattern := `/\A(?:` + strings.Join(names, "|") + `)\z/u`
		substitutions = append(substitutions, map[string]string{
			"file":   templating.ShellEscape(file),
			"filter": templating.ShellEscape(pattern),
		})
	}
	return substitutions, nil
}
