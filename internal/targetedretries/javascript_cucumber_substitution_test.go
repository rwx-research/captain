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

var _ = Describe("JavaScriptCucumberSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.JavaScriptCucumberSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.JavaScriptCucumberSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/cucumber-js.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.JavaScriptCucumberJSONParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(test v1.Test) bool { return true },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			return substitutions[i]["scenarios"] < substitutions[j]["scenarios"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.JavaScriptCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.JavaScriptCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("npx cucumber-js")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.JavaScriptCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("npx cucumber-js {{ file }} {{ scenarios }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without a scenarios placeholder", func() {
			substitution := targetedretries.JavaScriptCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("npx cucumber-js {{ file }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only a scenarios placeholder", func() {
			substitution := targetedretries.JavaScriptCucumberSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("npx cucumber-js {{ scenarios }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the shell escaped element start identifiers", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("npx cucumber-js {{ scenarios }}")
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "file1 with ' single quotes"
			file2 := "file2"
			file3 := "file3"
			file4 := "file4"
			file5 := "file5"
			file6 := "file6"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Location: &v1.Location{File: file1},
						Attempt: v1.TestAttempt{
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Location: &v1.Location{File: file2},
						Attempt: v1.TestAttempt{
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file3},
						Attempt: v1.TestAttempt{
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file4},
						Attempt: v1.TestAttempt{
							Status: v1.NewPendedTestStatus(nil),
						},
					},
					{
						Location: &v1.Location{File: file5},
						Attempt: v1.TestAttempt{
							Status: v1.NewSuccessfulTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file6},
						Attempt: v1.TestAttempt{
							Status: v1.NewSkippedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.JavaScriptCucumberSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return true },
			)).To(Equal(
				[]map[string]string{
					{
						"scenarios": `'file1 with '"'"' single quotes' ` +
							`'file2' ` +
							`'file3'`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("npx cucumber-js {{ scenarios }}")
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "file1 with ' single quotes"
			file2 := "file2"
			file3 := "file3"
			file4 := "file4"
			file5 := "file5"
			file6 := "file6"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Location: &v1.Location{File: file1},
						Attempt: v1.TestAttempt{
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Location: &v1.Location{File: file2},
						Attempt: v1.TestAttempt{
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file3},
						Attempt: v1.TestAttempt{
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file4},
						Attempt: v1.TestAttempt{
							Status: v1.NewPendedTestStatus(nil),
						},
					},
					{
						Location: &v1.Location{File: file5},
						Attempt: v1.TestAttempt{
							Status: v1.NewSuccessfulTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file6},
						Attempt: v1.TestAttempt{
							Status: v1.NewSkippedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.JavaScriptCucumberSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"scenarios": `'file1 with '"'"' single quotes'`,
					},
				},
			))
		})
	})
})
