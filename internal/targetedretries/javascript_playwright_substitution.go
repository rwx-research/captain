package targetedretries

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type JavaScriptPlaywrightSubstitution struct{}

func (s JavaScriptPlaywrightSubstitution) Example() string {
	return "npx playwright test {{ tests }} --project '{{ project }}'"
}

func (s JavaScriptPlaywrightSubstitution) ValidateTemplate(compiledTemplate templating.CompiledTemplate) error {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 0 {
		return errors.NewInputError(
			"Retrying Playwright requires a template with the 'tests' and 'project' keywords " +
				"or a template with the 'file', 'project', and 'grep' keywords; no keywords were found",
		)
	}

	if len(keywords) != 2 && len(keywords) != 3 {
		return errors.NewInputError(
			"Retrying Playwright requires a template with the 'tests' and 'project' keywords "+
				"or a template with the 'file', 'project', and 'grep' keywords; these were found: %v",
			strings.Join(keywords, ", "),
		)
	}

	sort.SliceStable(keywords, func(i, j int) bool {
		return keywords[i] < keywords[j]
	})

	if len(keywords) == 2 {
		if keywords[0] == "project" && keywords[1] == "tests" {
			return nil
		}

		return errors.NewInputError(
			"Retrying Playwright requires a template with the 'tests' and 'project' keywords "+
				"or a template with the 'file', 'project', and 'grep' keywords; "+
				"'%v' and '%v' were found instead",
			keywords[0],
			keywords[1],
		)
	}

	// else, len(keywords) == 3

	if keywords[0] == "file" && keywords[1] == "grep" && keywords[2] == "project" {
		return nil
	}

	return errors.NewInputError(
		"Retrying Playwright requires a template with the 'tests' and 'project' keywords "+
			"or a template with the 'file', 'project', and 'grep' keywords; "+
			"'%v', '%v', and '%v' were found instead",
		keywords[0],
		keywords[1],
		keywords[2],
	)
}

func (s JavaScriptPlaywrightSubstitution) SubstitutionsFor(
	compiledTemplate templating.CompiledTemplate,
	testResults v1.TestResults,
	filter func(v1.Test) bool,
) ([]map[string]string, error) {
	keywords := compiledTemplate.Keywords()

	if len(keywords) == 2 {
		// tests and project

		testsByProject := map[string][]string{}
		testsSeenByProject := map[string]map[string]struct{}{}

		for _, test := range testResults.Tests {
			if !filter(test) {
				continue
			}

			project := templating.ShellEscape(test.Attempt.Meta["project"].(string))
			location, _ := playwrightRetryLocation(test)
			test := templating.ShellEscape(location.String())

			if _, ok := testsSeenByProject[project]; !ok {
				testsSeenByProject[project] = map[string]struct{}{}
			}
			if _, ok := testsByProject[project]; !ok {
				testsByProject[project] = make([]string, 0)
			}

			if _, ok := testsSeenByProject[project][test]; ok {
				continue
			}

			testsByProject[project] = append(testsByProject[project], test)
			testsSeenByProject[project][test] = struct{}{}
		}

		substitutions := make([]map[string]string, 0, len(testsByProject))
		for project, tests := range testsByProject {
			substitutions = append(substitutions, map[string]string{
				"project": project,
				"tests":   strings.Join(tests, " "),
			})
		}

		return substitutions, nil
	}

	// else, len(keywords) == 3
	// file, project, and grep

	testsByFileByProject := map[string]map[string][]string{}
	testsSeenByFileByProject := map[string]map[string]map[string]struct{}{}
	serialLocationsByProject := map[string][]string{}
	serialLocationsSeenByProject := map[string]map[string]struct{}{}

	for _, test := range testResults.Tests {
		if !filter(test) {
			continue
		}

		project := templating.ShellEscape(test.Attempt.Meta["project"].(string))
		location, serial := playwrightRetryLocation(test)
		if serial {
			file := templating.ShellEscape(location.String())
			if _, ok := serialLocationsSeenByProject[project]; !ok {
				serialLocationsSeenByProject[project] = map[string]struct{}{}
			}
			if _, ok := serialLocationsSeenByProject[project][file]; ok {
				continue
			}

			serialLocationsByProject[project] = append(serialLocationsByProject[project], file)
			serialLocationsSeenByProject[project][file] = struct{}{}
			continue
		}

		file := templating.ShellEscape(test.Location.File)
		if _, ok := testsSeenByFileByProject[project]; !ok {
			testsSeenByFileByProject[project] = map[string]map[string]struct{}{}
		}
		if _, ok := testsSeenByFileByProject[project][file]; !ok {
			testsSeenByFileByProject[project][file] = map[string]struct{}{}
		}
		if _, ok := testsByFileByProject[project]; !ok {
			testsByFileByProject[project] = map[string][]string{}
		}
		if _, ok := testsByFileByProject[project][file]; !ok {
			testsByFileByProject[project][file] = make([]string, 0)
		}

		formattedName := templating.ShellEscape(templating.RegexpEscape(test.Name))
		if _, ok := testsSeenByFileByProject[project][file][formattedName]; ok {
			continue
		}

		testsByFileByProject[project][file] = append(testsByFileByProject[project][file], formattedName)
		testsSeenByFileByProject[project][file][formattedName] = struct{}{}
	}

	substitutions := make([]map[string]string, 0)
	for project, testsByFile := range testsByFileByProject {
		for file, tests := range testsByFile {
			substitutions = append(substitutions, map[string]string{
				"project": project,
				"file":    file,
				// Playwright matches grep against "project file describe title tags", and
				// keeps the leading space when the project name is empty.
				"grep": fmt.Sprintf(
					`^%v %v (%v)(?: @[^ ]*)*$`,
					templating.RegexpEscape(project),
					templating.RegexpEscape(file),
					strings.Join(tests, "|"),
				),
			})
		}
	}
	for project, locations := range serialLocationsByProject {
		for _, location := range locations {
			substitutions = append(substitutions, map[string]string{
				"project": project,
				"file":    location,
				"grep":    ".*",
			})
		}
	}

	return substitutions, nil
}

func playwrightRetryLocation(test v1.Test) (*v1.Location, bool) {
	annotations, ok := test.Attempt.Meta["annotations"].([]parsing.JavaScriptPlaywrightAnnotation)
	if !ok {
		return test.Location, false
	}

	for _, annotation := range annotations {
		if annotation.Type != "rwx:serial" || annotation.Location == nil ||
			annotation.Location.File == "" || annotation.Location.Line < 0 {
			continue
		}

		location := &v1.Location{File: annotation.Location.File}
		if annotation.Location.Line == 0 {
			return location, true
		}

		line := annotation.Location.Line
		location.Line = &line
		return location, true
	}

	return test.Location, false
}
