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

var _ = Describe("RubyCucumberSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.RubyCucumberSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.RubyCucumberSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/cucumber/failing.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.RubyCucumberParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(test v1.Test) bool { return true },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			return substitutions[i]["examples"] < substitutions[j]["examples"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.RubyCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.RubyCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("bundle exec cucumber")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.RubyCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("bundle exec cucumber {{ file }} {{ examples }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without a examples placeholder", func() {
			substitution := targetedretries.RubyCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("bundle exec cucumber {{ file }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only a examples placeholder", func() {
			substitution := targetedretries.RubyCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("bundle exec cucumber {{ examples }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the shell escaped element start identifiers", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("bundle exec cucumber {{ examples }}")
			Expect(compileErr).NotTo(HaveOccurred())

			elementStart1 := "elementStart1 with ' single quotes"
			elementStart2 := "elementStart2"
			elementStart3 := "elementStart3"
			elementStart4 := "elementStart4"
			elementStart5 := "elementStart5"
			elementStart6 := "elementStart6"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart2},
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart3},
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart4},
							Status: v1.NewPendedTestStatus(nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart5},
							Status: v1.NewSuccessfulTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart6},
							Status: v1.NewSkippedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.RubyCucumberSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return true },
			)).To(Equal(
				[]map[string]string{
					{
						"examples": `'elementStart1 with '"'"' single quotes' ` +
							`'elementStart2' ` +
							`'elementStart3'`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("bundle exec cucumber {{ examples }}")
			Expect(compileErr).NotTo(HaveOccurred())

			elementStart1 := "elementStart1 with ' single quotes"
			elementStart2 := "elementStart2"
			elementStart3 := "elementStart3"
			elementStart4 := "elementStart4"
			elementStart5 := "elementStart5"
			elementStart6 := "elementStart6"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart2},
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart3},
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart4},
							Status: v1.NewPendedTestStatus(nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart5},
							Status: v1.NewSuccessfulTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"elementStart": elementStart6},
							Status: v1.NewSkippedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.RubyCucumberSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"examples": `'elementStart1 with '"'"' single quotes'`,
					},
				},
			))
		})
	})
})
