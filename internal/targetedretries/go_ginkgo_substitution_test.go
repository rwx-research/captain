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

var _ = Describe("GoGinkgoSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.GoGinkgoSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.GoGinkgoSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/ginkgo.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.GoGinkgoParser{}.Parse(fixture)
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
			substitution := targetedretries.GoGinkgoSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.GoGinkgoSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("ginkgo run ./...")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.GoGinkgoSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("ginkgo run {{ file }} {{ tests }} ./...")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without a tests placeholder", func() {
			substitution := targetedretries.GoGinkgoSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("ginkgo run {{ file }} ./...")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only a tests placeholder", func() {
			substitution := targetedretries.GoGinkgoSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("ginkgo run {{ tests }} ./...")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the shell escaped test locations", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("ginkgo run {{ tests }} ./...")
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "/path/to/file with spaces.go"
			file2 := "/path/to/filewith'.go"
			file3 := "/path/to/otherfile.go"
			line1 := 1
			line2 := 100
			line3 := 1249
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Location: &v1.Location{File: file1, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Location: &v1.Location{File: file2, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line3},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)},
					},
					{
						Location: &v1.Location{File: file2, Line: &line3},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Location: &v1.Location{File: file1, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.GoGinkgoSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
			)).To(Equal(
				[]map[string]string{
					{
						"tests": `--focus-file '/path/to/file with spaces.go:1' ` +
							`--focus-file '/path/to/filewith'"'"'.go:100' ` +
							`--focus-file '/path/to/otherfile.go:1249'`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("ginkgo run {{ tests }} ./...")
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "/path/to/file with spaces.go"
			file2 := "/path/to/filewith'.go"
			file3 := "/path/to/otherfile.go"
			line1 := 1
			line2 := 100
			line3 := 1249
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Location: &v1.Location{File: file1, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Location: &v1.Location{File: file2, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line3},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)},
					},
					{
						Location: &v1.Location{File: file2, Line: &line3},
						Attempt:  v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
					},
					{
						Location: &v1.Location{File: file1, Line: &line2},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Location: &v1.Location{File: file3, Line: &line1},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.GoGinkgoSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"tests": "--focus-file '/path/to/file with spaces.go:1'",
					},
				},
			))
		})
	})
})
