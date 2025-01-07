package targetedretries_test

import (
	"os"
	"sort"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptCypressSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.JavaScriptCypressSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.JavaScriptCypressSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/cypress.xml")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(_ v1.Test) bool { return true },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["spec"] != substitutions[j]["spec"] {
				return substitutions[i]["spec"] < substitutions[j]["spec"]
			}

			return substitutions[i]["grep"] < substitutions[j]["grep"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.JavaScriptCypressSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.JavaScriptCypressSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("npx cypress run")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with additional placeholders", func() {
			substitution := targetedretries.JavaScriptCypressSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx cypress run --spec '{{ spec }}' --env {{ grep }} {{ foo }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders", func() {
			substitution := targetedretries.JavaScriptCypressSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx cypress run --spec '{{ wat }}' --env {{ foo }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without the spec placeholder", func() {
			substitution := targetedretries.JavaScriptCypressSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx cypress run --env {{ grep }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only the spec placeholder", func() {
			substitution := targetedretries.JavaScriptCypressSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx cypress run --spec '{{ spec }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid for a template with the spec and grep placeholders", func() {
			substitution := targetedretries.JavaScriptCypressSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx cypress run --spec '{{ spec }}' --env {{ grep }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns tests grouped by spec", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx cypress run --spec '{{ spec }}' --env {{ grep }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			spec1 := "spec1.js"
			spec2 := "path/to/spec with ' in it.js"

			name1 := "name1"
			name2 := "name2 with ' in it"
			name3 := "name3"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name:     name1,
						Location: &v1.Location{File: spec1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Name:     name2,
						Location: &v1.Location{File: spec2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Name:     name3,
						Location: &v1.Location{File: spec1},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Name:     name1,
						Location: &v1.Location{File: spec2},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Name:     name2,
						Location: &v1.Location{File: spec1},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Name:     name3,
						Location: &v1.Location{File: spec2},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptCypressSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(_ v1.Test) bool { return true },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["spec"] < substitutions[j]["spec"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"spec": `path/to/spec with '"'"' in it.js`,
						"grep": `grep='name2 with '"'"' in it'`,
					},
					{
						"spec": "spec1.js",
						"grep": "grep='name1; name3'",
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx cypress run --spec '{{ spec }}' --env {{ grep }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			spec1 := "spec1.js"
			spec2 := "path/to/spec with ' in it.js"

			name1 := "name1"
			name2 := "name2 with ' in it"
			name3 := "name3"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name:     name1,
						Location: &v1.Location{File: spec1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Name:     name2,
						Location: &v1.Location{File: spec2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Name:     name3,
						Location: &v1.Location{File: spec1},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Name:     name1,
						Location: &v1.Location{File: spec2},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Name:     name2,
						Location: &v1.Location{File: spec1},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Name:     name3,
						Location: &v1.Location{File: spec2},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptCypressSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["spec"] < substitutions[j]["spec"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"spec": "spec1.js",
						"grep": "grep='name1'",
					},
				},
			))
		})
	})
})
