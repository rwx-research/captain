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

var _ = Describe("RubyRSpecSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.RubyRSpecSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.RubyRSpecSubstitution{}
		compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/rspec.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.RubyRSpecParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(test v1.Test) bool { return true },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			return substitutions[i]["tests"] < substitutions[j]["tests"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.RubyRSpecSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.RubyRSpecSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bundle exec rspec")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.RubyRSpecSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bundle exec rspec {{ file }} {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without a tests placeholder", func() {
			substitution := targetedretries.RubyRSpecSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bundle exec rspec {{ file }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only a tests placeholder", func() {
			substitution := targetedretries.RubyRSpecSubstitution{}
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bundle exec rspec {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the shell escaped test identifiers", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bundle exec rspec {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			id1 := "/path/to/file with spaces.rb:10"
			id2 := "/path/to/filewith'.rb:15"
			id3 := "/path/to/otherfile.rb:[1:2:2:5]"
			id4 := "/path/to/otherfile.rb:[1:3:2:5]"
			id5 := "/path/to/otherfile.rb:[1:4:2:5]"
			id6 := "/path/to/otherfile.rb:[1:5:2:5]"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{ID: &id1, Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
					{ID: &id2, Attempt: v1.TestAttempt{Status: v1.NewCanceledTestStatus()}},
					{ID: &id3, Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus()}},
					{ID: &id4, Attempt: v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)}},
					{ID: &id5, Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
					{ID: &id6, Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
				},
			}

			substitution := targetedretries.RubyRSpecSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return true },
			)).To(Equal(
				[]map[string]string{
					{
						"tests": `'/path/to/file with spaces.rb:10' '/path/to/filewith'"'"'.rb:15' ` +
							`'/path/to/otherfile.rb:[1:2:2:5]'`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := targetedretries.CompileTemplate("bundle exec rspec {{ tests }}")
			Expect(compileErr).NotTo(HaveOccurred())

			id1 := "/path/to/file with spaces.rb:10"
			id2 := "/path/to/filewith'.rb:15"
			id3 := "/path/to/otherfile.rb:[1:2:2:5]"
			id4 := "/path/to/otherfile.rb:[1:3:2:5]"
			id5 := "/path/to/otherfile.rb:[1:4:2:5]"
			id6 := "/path/to/otherfile.rb:[1:5:2:5]"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{ID: &id1, Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
					{ID: &id2, Attempt: v1.TestAttempt{Status: v1.NewCanceledTestStatus()}},
					{ID: &id3, Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus()}},
					{ID: &id4, Attempt: v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)}},
					{ID: &id5, Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
					{ID: &id6, Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
				},
			}

			substitution := targetedretries.RubyRSpecSubstitution{}
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
	})
})
