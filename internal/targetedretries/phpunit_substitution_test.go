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

var _ = Describe("PHPUnitSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.PHPUnitSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.PHPUnitSubstitution{}
		compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/phpunit.xml")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.PHPUnitParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(test v1.Test) bool { return true },
		)
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["file"] != substitutions[j]["file"] {
				return substitutions[i]["file"] < substitutions[j]["file"]
			}

			return substitutions[i]["filter"] < substitutions[j]["filter"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.PHPUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.PHPUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("vendor/bin/phpunit")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too few placeholders", func() {
			substitution := targetedretries.PHPUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"vendor/bin/phpunit '{{ file }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with additional placeholders", func() {
			substitution := targetedretries.PHPUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"vendor/bin/phpunit --filter '{{ filter }}' '{{ file }}' {{ foo }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders", func() {
			substitution := targetedretries.PHPUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"vendor/bin/phpunit --filter '{{ foo }}' '{{ bar }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with the file and grep placeholders", func() {
			substitution := targetedretries.PHPUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"vendor/bin/phpunit --filter '{{ filter }}' '{{ file }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns a substitution per file/test", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"vendor/bin/phpunit '{{ file }}' --grep '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "path/to/file1.php"
			file2 := "path/to/file with '.php"

			name1 := "some_test_name"
			name2 := "other_test_name"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Location: &v1.Location{File: file1},
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"name": name1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Location: &v1.Location{File: file2},
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"name": name1},
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file1},
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"name": name2},
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file2},
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"name": name2},
							Status: v1.NewPendedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.PHPUnitSubstitution{}
			substitutions := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return true },
			)
			sort.SliceStable(substitutions, func(i int, j int) bool {
				if substitutions[i]["file"] != substitutions[j]["file"] {
					return substitutions[i]["file"] < substitutions[j]["file"]
				}

				return substitutions[i]["filter"] < substitutions[j]["filter"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file":   `path/to/file with '"'"'.php`,
						"filter": "some_test_name",
					},
					{
						"file":   "path/to/file1.php",
						"filter": "other_test_name",
					},
					{
						"file":   "path/to/file1.php",
						"filter": "some_test_name",
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate(
				"vendor/bin/phpunit '{{ file }}' --grep '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "path/to/file1.php"
			file2 := "path/to/file with '.php"

			name1 := "some_test_name"
			name2 := "other_test_name"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Location: &v1.Location{File: file1},
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"name": name1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Location: &v1.Location{File: file2},
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"name": name1},
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file1},
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"name": name2},
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Location: &v1.Location{File: file2},
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"name": name2},
							Status: v1.NewPendedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.PHPUnitSubstitution{}
			substitutions := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			sort.SliceStable(substitutions, func(i int, j int) bool {
				if substitutions[i]["file"] != substitutions[j]["file"] {
					return substitutions[i]["file"] < substitutions[j]["file"]
				}

				return substitutions[i]["filter"] < substitutions[j]["filter"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file":   "path/to/file1.php",
						"filter": "some_test_name",
					},
				},
			))
		})
	})
})
