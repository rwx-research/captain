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

var _ = Describe("ElixirExUnitSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.ElixirExUnitSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.ElixirExUnitSubstitution{}
		compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/exunit.xml")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.ElixirExUnitParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(test v1.Test) bool { return true },
		)
		sort.SliceStable(substitutions, func(i int, j int) bool {
			return substitutions[i]["tests"] < substitutions[j]["tests"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.ElixirExUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.ElixirExUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("mix test")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.ElixirExUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("mix test {{ file }} {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without a tests placeholder", func() {
			substitution := targetedretries.ElixirExUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("mix test {{ file }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only a tests placeholder", func() {
			substitution := targetedretries.ElixirExUnitSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("mix test {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the shell escaped test identifiers", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate("mix test {{ tests }}")
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
						Location: &v1.Location{File: file1, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line2},
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

			substitution := targetedretries.ElixirExUnitSubstitution{}
			substitutions := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return true },
			)
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["tests"] < substitutions[j]["tests"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"tests": "'/path/to/file with spaces.rb:10:20'",
					},
					{
						"tests": `'/path/to/filewith'"'"'.rb:10'`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate("mix test {{ tests }}")
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
						Location: &v1.Location{File: file1, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line2},
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

			substitution := targetedretries.ElixirExUnitSubstitution{}
			substitutions := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["tests"] < substitutions[j]["tests"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"tests": "'/path/to/file with spaces.rb:10'",
					},
				},
			))
		})
	})
})
