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

var _ = Describe("RubyMinitestSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.RubyMinitestSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file for Rails", func() {
		substitution := targetedretries.RubyMinitestSubstitution{}
		compiledTemplate, compileErr := targetedretries.CompileTemplate("bin/rails test {{ tests }}")
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/minitest.xml")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.RubyMinitestParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(test v1.Test) bool { return true },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["file"] != substitutions[j]["file"] {
				return substitutions[i]["file"] < substitutions[j]["file"]
			}

			return substitutions[i]["name"] < substitutions[j]["name"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	It("works with a real file for non-Rails", func() {
		substitution := targetedretries.RubyMinitestSubstitution{}
		compiledTemplate, compileErr := targetedretries.CompileTemplate("ruby -I test '{{ file }}' --name '{{ name }}'")
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/minitest.xml")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.RubyMinitestParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(test v1.Test) bool { return true },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["file"] != substitutions[j]["file"] {
				return substitutions[i]["file"] < substitutions[j]["file"]
			}

			return substitutions[i]["name"] < substitutions[j]["name"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.RubyMinitestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.RubyMinitestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bin/rails test")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.RubyMinitestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bin/rails test {{ file }} {{ test }} {{ wat }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with one placeholder that isn't tests", func() {
			substitution := targetedretries.RubyMinitestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bin/rails test {{ file }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with one placeholder that is tests", func() {
			substitution := targetedretries.RubyMinitestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bin/rails test {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is invalid for a template with two placeholder that are both incorrect", func() {
			substitution := targetedretries.RubyMinitestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("ruby -I test '{{ foo }}' --name '{{ bar }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with two placeholder where one is incorrect", func() {
			substitution := targetedretries.RubyMinitestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("ruby -I test '{{ file }}' --name '{{ bar }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with a file and name placeholder", func() {
			substitution := targetedretries.RubyMinitestSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("ruby -I test '{{ file }}' --name '{{ name }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the shell escaped test locations for the tests keyword", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bin/rails test {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "/path/to/file with spaces.rb"
			file2 := "/path/to/filewith'.rb"
			file3 := "/path/to/file.rb"

			line1 := 10
			line2 := 20

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Location: &v1.Location{File: file1, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Location: &v1.Location{File: file2, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Location: &v1.Location{File: file1, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Location: &v1.Location{File: file2, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.RubyMinitestSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return true },
			)).To(Equal(
				[]map[string]string{
					{
						"tests": `'/path/to/file with spaces.rb:10' ` +
							`'/path/to/filewith'"'"'.rb:10' ` +
							`'/path/to/file.rb:20'`,
					},
				},
			))
		})

		It("filters the tests for the tests keyword with the provided function", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bin/rails test {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "/path/to/file with spaces.rb"
			file2 := "/path/to/filewith'.rb"
			file3 := "/path/to/file.rb"

			line1 := 10
			line2 := 20

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Location: &v1.Location{File: file1, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Location: &v1.Location{File: file2, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Location: &v1.Location{File: file1, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Location: &v1.Location{File: file2, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.RubyMinitestSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"tests": "'/path/to/file with spaces.rb:10'",
					},
				},
			))
		})

		It("returns the individual escaped files and names", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate("ruby -I test '{{ file }}' --name '{{ name }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "/path/to/file with spaces.rb"
			file2 := "/path/to/filewith'.rb"
			file3 := "/path/to/file.rb"

			lineage1 := []string{"classname", "test_method_name_1"}
			lineage2 := []string{"classname", "test_method_name_2"}

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
						Location: &v1.Location{File: file3},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Lineage:  lineage2,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Lineage:  lineage2,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Lineage:  lineage1,
						Location: &v1.Location{File: file3},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.RubyMinitestSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return true },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["file"] < substitutions[j]["file"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file": "/path/to/file with spaces.rb",
						"name": "test_method_name_1",
					},
					{
						"file": "/path/to/file.rb",
						"name": "test_method_name_2",
					},
					{
						"file": `/path/to/filewith'"'"'.rb`,
						"name": "test_method_name_1",
					},
				},
			))
		})

		It("filters the tests for the file/name keywords with the provided function", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate("ruby -I test '{{ file }}' --name '{{ name }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "/path/to/file with spaces.rb"
			file2 := "/path/to/filewith'.rb"
			file3 := "/path/to/file.rb"

			lineage1 := []string{"classname", "test_method_name_1"}
			lineage2 := []string{"classname", "test_method_name_2"}

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
						Location: &v1.Location{File: file3},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Lineage:  lineage2,
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Lineage:  lineage2,
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Lineage:  lineage1,
						Location: &v1.Location{File: file3},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.RubyMinitestSubstitution{}
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
						"file": "/path/to/file with spaces.rb",
						"name": "test_method_name_1",
					},
				},
			))
		})
	})
})
