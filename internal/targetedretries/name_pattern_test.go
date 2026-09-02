package targetedretries_test

import (
	"regexp"
	"strings"

	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Each framework matches its retry pattern against a different string. These cases record
// that string so the pattern is checked the way its runner would use it. The Playwright
// composition is verified against playwright 1.62.1 with `playwright test --list --grep`.
type namePatternCase struct {
	framework  string
	template   string
	patternKey string
	// retried names fail; siblings pass and must not be selected by the pattern.
	retried  []string
	siblings []string
	newTest  func(name string, failed bool) v1.Test
	// target composes the string the runner matches the pattern against.
	target func(name string) string
}

var _ = Describe("targeted retry name patterns", func() {
	const file = "path/to/example_test.js"
	const project = "chromium"
	const goPackage = "github.com/rwx-research/captain-cli/internal/example"
	suite := "outer suite"

	status := func(failed bool) v1.TestStatus {
		if failed {
			return v1.NewFailedTestStatus(nil, nil, nil)
		}

		return v1.NewSuccessfulTestStatus()
	}

	// A test whose name is a strict prefix of a sibling, in both shapes a suite hits it:
	// an appended digit (jest's `.each` index placeholders) and an appended word.
	jsRetried := []string{"rejects bad input 1", "accepts valid input"}
	jsSiblings := []string{
		"rejects bad input 10",         // digit appended, no separator
		"rejects bad input 1 extended", // word appended after a space
		"also rejects bad input 1",     // the retried name is a suffix of this one
		"accepts valid input twice",
	}

	lineageTest := func(name string, failed bool) v1.Test {
		lineage := []string{suite, name}
		return v1.Test{
			Name:     strings.Join(lineage, " "),
			Lineage:  lineage,
			Location: &v1.Location{File: file},
			Attempt:  v1.TestAttempt{Status: status(failed)},
		}
	}
	lineageTarget := func(name string) string {
		return strings.Join([]string{suite, name}, " ")
	}

	cases := []namePatternCase{
		{
			framework:  "jest",
			template:   "npx jest --testPathPattern '{{ testPathPattern }}' --testNamePattern '{{ testNamePattern }}'",
			patternKey: "testNamePattern",
			retried:    jsRetried,
			siblings:   jsSiblings,
			newTest:    lineageTest,
			target:     lineageTarget,
		},
		{
			framework:  "vitest",
			template:   "npx vitest run '{{ file }}' --testNamePattern '{{ testNamePattern }}'",
			patternKey: "testNamePattern",
			retried:    jsRetried,
			siblings:   jsSiblings,
			newTest:    lineageTest,
			target:     lineageTarget,
		},
		{
			framework:  "bun",
			template:   "bun test '{{ file }}' --test-name-pattern '{{ testNamePattern }}'",
			patternKey: "testNamePattern",
			retried:    jsRetried,
			siblings:   jsSiblings,
			newTest:    lineageTest,
			target:     lineageTarget,
		},
		{
			framework:  "mocha",
			template:   "npx mocha '{{ file }}' --grep '{{ grep }}'",
			patternKey: "grep",
			retried:    jsRetried,
			siblings:   jsSiblings,
			newTest:    lineageTest,
			target:     lineageTarget,
		},
		{
			framework:  "testcafe",
			template:   "npx testcafe chrome '{{ file }}' --test-grep '{{ grep }}'",
			patternKey: "grep",
			retried:    jsRetried,
			siblings:   jsSiblings,
			newTest:    lineageTest,
			// testcafe greps the test name on its own, without its fixture.
			target: func(name string) string { return name },
		},
		{
			framework:  "go test",
			template:   "go test '{{ package }}' -run '{{ run }}'",
			patternKey: "run",
			retried:    []string{"TestCase1", "TestOther"},
			siblings:   []string{"TestCase10", "TestCase1Extended", "TestNotTestCase1", "TestOtherThing"},
			newTest: func(name string, failed bool) v1.Test {
				return v1.Test{
					Name: name,
					Attempt: v1.TestAttempt{
						Status: status(failed),
						Meta:   map[string]any{"package": goPackage},
					},
				}
			},
			target: func(name string) string { return name },
		},
		{
			framework:  "playwright",
			template:   "npx playwright test '{{ file }}' --project '{{ project }}' --grep '{{ grep }}'",
			patternKey: "grep",
			retried:    jsRetried,
			siblings:   jsSiblings,
			newTest: func(name string, failed bool) v1.Test {
				test := lineageTest(name, failed)
				test.Attempt.Meta = map[string]any{"project": project}
				return test
			},
			// playwright greps "project file describe title tags" joined by spaces.
			target: func(name string) string {
				return strings.Join([]string{project, file, suite, name}, " ")
			},
		},
	}

	for _, testCase := range cases {
		testCase := testCase

		Describe(testCase.framework, func() {
			pattern := func() *regexp.Regexp {
				compiledTemplate, err := templating.CompileTemplate(testCase.template)
				Expect(err).NotTo(HaveOccurred())

				substitution, ok := targetedretries.SubstitutionsByFramework[frameworkFor(testCase.framework)]
				Expect(ok).To(BeTrue(), "no substitution registered for %v", testCase.framework)

				tests := make([]v1.Test, 0, len(testCase.retried)+len(testCase.siblings))
				for _, name := range testCase.retried {
					tests = append(tests, testCase.newTest(name, true))
				}
				for _, name := range testCase.siblings {
					tests = append(tests, testCase.newTest(name, false))
				}

				substitutions, err := substitution.SubstitutionsFor(
					compiledTemplate,
					v1.TestResults{Tests: tests},
					func(test v1.Test) bool { return test.Attempt.Status.ImpliesFailure() },
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(substitutions).To(HaveLen(1))

				// The pattern reaches the runner through a single-quoted shell argument, so
				// undo the shell escaping to recover the regexp the runner compiles.
				source := strings.ReplaceAll(substitutions[0][testCase.patternKey], `'"'"'`, "'")
				compiled, err := regexp.Compile(source)
				Expect(err).NotTo(HaveOccurred(), "pattern is not a valid regexp: %v", source)

				return compiled
			}

			It("selects every retried test", func() {
				compiled := pattern()

				for _, name := range testCase.retried {
					Expect(compiled.MatchString(testCase.target(name))).To(
						BeTrue(),
						"pattern %v did not select retried test %q",
						compiled, testCase.target(name),
					)
				}
			})

			It("selects no test that merely extends or contains a retried name", func() {
				compiled := pattern()

				for _, name := range testCase.siblings {
					Expect(compiled.MatchString(testCase.target(name))).To(
						BeFalse(),
						"pattern %v over-matched sibling test %q",
						compiled, testCase.target(name),
					)
				}
			})
		})
	}
})

func frameworkFor(name string) v1.Framework {
	switch name {
	case "jest":
		return v1.JavaScriptJestFramework
	case "vitest":
		return v1.JavaScriptVitestFramework
	case "bun":
		return v1.JavaScriptBunFramework
	case "mocha":
		return v1.JavaScriptMochaFramework
	case "testcafe":
		return v1.JavaScriptTestCafeFramework
	case "go test":
		return v1.GoTestFramework
	case "playwright":
		return v1.JavaScriptPlaywrightFramework
	}

	return v1.Framework{}
}
