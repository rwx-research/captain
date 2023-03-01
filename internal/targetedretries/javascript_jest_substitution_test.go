package targetedretries_test

import (
	"os"
	"sort"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptJestSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.JavaScriptJestSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.JavaScriptJestSubstitution{}
		compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/jest.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.JavaScriptJestParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(test v1.Test) bool { return true },
		)
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["testPathPattern"] != substitutions[j]["testPathPattern"] {
				return substitutions[i]["testPathPattern"] < substitutions[j]["testPathPattern"]
			}

			return substitutions[i]["testNamePattern"] < substitutions[j]["testNamePattern"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.JavaScriptJestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.JavaScriptJestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("npx jest")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too few placeholders", func() {
			substitution := targetedretries.JavaScriptJestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"npx jest --testPathPattern '{{ testPathPattern }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with additional placeholders", func() {
			substitution := targetedretries.JavaScriptJestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"npx jest --testPathPattern '{{ testPathPattern }}' --testNamePattern '{{ testNamePattern }}' {{ foo }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders", func() {
			substitution := targetedretries.JavaScriptJestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"npx jest --testPathPattern '{{ wat }}' --testNamePattern '{{ foo }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with the testPathPattern and testNamePattern placeholders", func() {
			substitution := targetedretries.JavaScriptJestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"npx jest --testPathPattern '{{ testPathPattern }}' --testNamePattern '{{ testNamePattern }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns tests grouped by file", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"npx jest --testPathPattern '{{ testPathPattern }}' --testNamePattern '{{ testNamePattern }}'",
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
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
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

			substitution := targetedretries.JavaScriptJestSubstitution{}
			substitutions := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return true },
			)
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["testPathPattern"] < substitutions[j]["testPathPattern"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"testPathPattern": `path/to/file with '"'"'.js`,
						"testNamePattern": `^name of describe test '"'"'one'"'"' \+ 1$`,
					},
					{
						"testPathPattern": "path/to/file1.js",
						"testNamePattern": `^name of describe test '"'"'one'"'"' \+ 1|name of describe test 2$`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"npx jest --testPathPattern '{{ testPathPattern }}' --testNamePattern '{{ testNamePattern }}'",
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
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
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

			substitution := targetedretries.JavaScriptJestSubstitution{}
			substitutions := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["testPathPattern"] < substitutions[j]["testPathPattern"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"testPathPattern": "path/to/file1.js",
						"testNamePattern": `^name of describe test '"'"'one'"'"' \+ 1$`,
					},
				},
			))
		})
	})
})
