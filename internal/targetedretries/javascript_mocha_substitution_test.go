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

var _ = Describe("JavaScriptMochaSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.JavaScriptMochaSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.JavaScriptMochaSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/mocha.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.JavaScriptMochaParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(_ v1.Test) bool { return true },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["file"] != substitutions[j]["file"] {
				return substitutions[i]["file"] < substitutions[j]["file"]
			}

			return substitutions[i]["grep"] < substitutions[j]["grep"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.JavaScriptMochaSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.JavaScriptMochaSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("npx mocha")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too few placeholders", func() {
			substitution := targetedretries.JavaScriptMochaSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx mocha '{{ file }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with additional placeholders", func() {
			substitution := targetedretries.JavaScriptMochaSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx mocha '{{ file }}' --grep '{{ grep }}' {{ foo }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders", func() {
			substitution := targetedretries.JavaScriptMochaSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx mocha '{{ wat }}' --grep '{{ foo }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with the file and grep placeholders", func() {
			substitution := targetedretries.JavaScriptMochaSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx mocha '{{ file }}' --grep '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns tests grouped by file", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx mocha '{{ file }}' --grep '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "path/to/file1.js"
			file2 := "path/to/file with '.js"

			name1 := "name of describe test 'one' + 1"
			name2 := "name of describe test 2"
			name3 := "name of describe test 3"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name:     name1,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Name:     name1,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Name:     name2,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Name:     name2,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Name:     name3,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Name:     name3,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptMochaSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(_ v1.Test) bool { return true },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["file"] < substitutions[j]["file"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file": `path/to/file with '"'"'.js`,
						"grep": `^name of describe test '"'"'one'"'"' \+ 1$`,
					},
					{
						"file": "path/to/file1.js",
						"grep": `^name of describe test '"'"'one'"'"' \+ 1|name of describe test 2$`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx mocha '{{ file }}' --grep '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "path/to/file1.js"
			file2 := "path/to/file with '.js"

			name1 := "name of describe test 'one' + 1"
			name2 := "name of describe test 2"
			name3 := "name of describe test 3"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name:     name1,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Name:     name1,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Name:     name2,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Name:     name2,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Name:     name3,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Name:     name3,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptMochaSubstitution{}
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
						"file": "path/to/file1.js",
						"grep": `^name of describe test '"'"'one'"'"' \+ 1$`,
					},
				},
			))
		})
	})
})
