package targetedretries_test

import (
	"sort"

	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptBunSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.JavaScriptBunSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.JavaScriptBunSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.JavaScriptBunSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("bun test")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too few placeholders", func() {
			substitution := targetedretries.JavaScriptBunSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"bun test '{{ file }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with additional placeholders", func() {
			substitution := targetedretries.JavaScriptBunSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"bun test '{{ file }}' --test-name-pattern '{{ testNamePattern }}' {{ foo }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders", func() {
			substitution := targetedretries.JavaScriptBunSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"bun test '{{ wat }}' --test-name-pattern '{{ foo }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with the file and testNamePattern placeholders", func() {
			substitution := targetedretries.JavaScriptBunSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"bun test '{{ file }}' --test-name-pattern '{{ testNamePattern }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns tests grouped by file", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"bun test '{{ file }}' --test-name-pattern '{{ testNamePattern }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "path/to/file1.js"
			file2 := "path/to/file with '.js"

			lineage1 := []string{"name of describe", "test 'one' + 1"}
			lineage2 := []string{"name of describe", "test 2"}
			lineage3 := []string{"name of describe", "test 3"}

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Lineage:  lineage1,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Lineage:  lineage1,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Lineage:  lineage2,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)},
					},
					{
						Lineage:  lineage2,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Lineage:  lineage3,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Lineage:  lineage3,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptBunSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["file"] < substitutions[j]["file"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file":            `path/to/file with '"'"'.js`,
						"testNamePattern": `^name of describe test '"'"'one'"'"' \+ 1$`,
					},
					{
						"file":            "path/to/file1.js",
						"testNamePattern": `^name of describe test '"'"'one'"'"' \+ 1|name of describe test 2$`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"bun test '{{ file }}' --test-name-pattern '{{ testNamePattern }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "path/to/file1.js"
			file2 := "path/to/file with '.js"

			lineage1 := []string{"name of describe", "test 'one' + 1"}
			lineage2 := []string{"name of describe", "test 2"}
			lineage3 := []string{"name of describe", "test 3"}

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Lineage:  lineage1,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Lineage:  lineage1,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Lineage:  lineage2,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)},
					},
					{
						Lineage:  lineage2,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Lineage:  lineage3,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Lineage:  lineage3,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptBunSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["file"] < substitutions[j]["file"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file":            "path/to/file1.js",
						"testNamePattern": `^name of describe test '"'"'one'"'"' \+ 1$`,
					},
				},
			))
		})
	})
})
