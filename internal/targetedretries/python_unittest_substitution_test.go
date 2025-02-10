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

var _ = Describe("PythonUnitTestSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.PythonUnitTestSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.PythonUnitTestSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/unittest.xml")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.PythonUnitTestParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			return substitutions[i]["tests"] < substitutions[j]["tests"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.PythonUnitTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.PythonUnitTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("unittest")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.PythonUnitTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("unittest {{ file }} {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without a tests placeholder", func() {
			substitution := targetedretries.PythonUnitTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("unittest {{ file }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only a tests placeholder", func() {
			substitution := targetedretries.PythonUnitTestSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("unittest {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the shell escaped test identifiers", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("unittest {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			name1 := "SomeClass.some_method"
			name2 := "Some ' Class.with_single_quotes"
			name3 := "some_method"
			name4 := "name4"
			name5 := "name5"
			name6 := "name6"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{Name: name1, Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
					{Name: name2, Attempt: v1.TestAttempt{Status: v1.NewCanceledTestStatus()}},
					{Name: name3, Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus()}},
					{Name: name4, Attempt: v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)}},
					{Name: name5, Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
					{Name: name6, Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
				},
			}

			substitution := targetedretries.PythonUnitTestSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
			)).To(Equal(
				[]map[string]string{
					{
						"tests": `'SomeClass.some_method' ` +
							`'Some '"'"' Class.with_single_quotes' ` +
							`'some_method'`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("unittest {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			name1 := "SomeClass.some_method"
			name2 := "Some ' Class.with_single_quotes"
			name3 := "some_method"
			name4 := "name4"
			name5 := "name5"
			name6 := "name6"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{Name: name1, Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
					{Name: name2, Attempt: v1.TestAttempt{Status: v1.NewCanceledTestStatus()}},
					{Name: name3, Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus()}},
					{Name: name4, Attempt: v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)}},
					{Name: name5, Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
					{Name: name6, Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
				},
			}

			substitution := targetedretries.PythonUnitTestSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"tests": "'SomeClass.some_method'",
					},
				},
			))
		})
	})
})
