package targetedretries_test

import (
	"os"
	"sort"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GoTestSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.GoTestSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.GoTestSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/go_test.jsonl")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.GoTestParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["package"] != substitutions[j]["package"] {
				return substitutions[i]["package"] < substitutions[j]["package"]
			}

			return substitutions[i]["run"] < substitutions[j]["run"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.GoTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.GoTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("go test")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too few placeholders", func() {
			substitution := targetedretries.GoTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("go test '{{ package }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with additional placeholders", func() {
			substitution := targetedretries.GoTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("go test '{{ package }}' -run '{{ run }}' {{ foo }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders", func() {
			substitution := targetedretries.GoTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("go test '{{ what }}' -run '{{ other }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with the package and run placeholders", func() {
			substitution := targetedretries.GoTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("go test '{{ package }}' -run '{{ run }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns tests grouped by package", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("go test '{{ package }}' -run '{{ run }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			package1 := "package1"
			package2 := "package2"

			name1 := "name1"
			name2 := "name2"
			name3 := "name3"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name: name1,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Name: name1,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package2},
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Name: name2,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Name: name2,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package2},
							Status: v1.NewPendedTestStatus(nil),
						},
					},
					{
						Name: name3,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewSuccessfulTestStatus(),
						},
					},
					{
						Name: name3,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package2},
							Status: v1.NewSkippedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.GoTestSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["package"] < substitutions[j]["package"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"package": "package1",
						"run":     "^name1|name2$",
					},
					{
						"package": "package2",
						"run":     "^name1$",
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("go test '{{ package }}' -run '{{ run }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			package1 := "package1"
			package2 := "package2"

			name1 := "name1"
			name2 := "name2"
			name3 := "name3"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name: name1,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Name: name1,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package2},
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Name: name2,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Name: name2,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package2},
							Status: v1.NewPendedTestStatus(nil),
						},
					},
					{
						Name: name3,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewSuccessfulTestStatus(),
						},
					},
					{
						Name: name3,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package2},
							Status: v1.NewSkippedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.GoTestSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["package"] < substitutions[j]["package"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"package": "package1",
						"run":     "^name1$",
					},
				},
			))
		})

		It("regex escapes the test name and shell escapes the package", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("go test '{{ package }}' -run '{{ run }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			package1 := "package1 with ' embedded"
			name1 := "name1 with + and . embedded"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name: name1,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
				},
			}

			substitution := targetedretries.GoTestSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(_ v1.Test) bool { return true },
			)).To(Equal(
				[]map[string]string{
					{
						"package": `package1 with '"'"' embedded`,
						"run":     `^name1 with \+ and \. embedded$`,
					},
				},
			))
		})

		It("adds entire table tests one time when it sees them", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("go test '{{ package }}' -run '{{ run }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			package1 := "package1"
			table1 := "Table/1"
			table2 := "Table/2"
			table3 := "Table/3"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name: table1,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Name: table2,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Name: table3,
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"package": package1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
				},
			}

			substitution := targetedretries.GoTestSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(_ v1.Test) bool { return true },
			)).To(Equal(
				[]map[string]string{
					{
						"package": "package1",
						"run":     "^Table$",
					},
				},
			))
		})
	})
})
