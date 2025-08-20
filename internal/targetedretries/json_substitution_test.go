package targetedretries_test

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JSONSubstitution", func() {
	var (
		mockFileSystem *mocks.FileSystem
		mockFile       *mocks.File
	)

	BeforeEach(func() {
		mockFileSystem = new(mocks.FileSystem)
		mockFile = new(mocks.File)
		mockFile.Builder = new(strings.Builder)
		mockFile.MockName = func() string {
			return "json-temp-file"
		}
		mockFileSystem.MockCreateTemp = func(_, _ string) (fs.File, error) {
			return mockFile, nil
		}
	})

	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
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
			func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(substitutions).To(HaveLen(1))
		cupaloy.SnapshotT(GinkgoT(), mockFile.Builder.String())
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			compiledTemplate, compileErr := templating.CompileTemplate("bin/your-script")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			compiledTemplate, compileErr := templating.CompileTemplate("bin/your-script {{ file }} {{ jsonFilePath }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without a jsonFilePath placeholder", func() {
			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			compiledTemplate, compileErr := templating.CompileTemplate("bin/your-script {{ file }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only a jsonFilePath placeholder", func() {
			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			compiledTemplate, compileErr := templating.CompileTemplate("bin/your-script {{ jsonFilePath }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the path of a temporary file", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("bin/your-script {{ jsonFilePath }}")
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
					{ID: &id3, Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)}},
					{ID: &id4, Attempt: v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)}},
					{ID: &id5, Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
					{ID: &id6, Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
				},
			}

			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
			)).To(Equal(
				[]map[string]string{
					{
						"jsonFilePath": "json-temp-file",
					},
				},
			))

			parsedTestResults := v1.TestResults{}
			decoder := json.NewDecoder(strings.NewReader(mockFile.Builder.String()))
			err := decoder.Decode(&parsedTestResults)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedTestResults.Summary.Tests).To(Equal(3))
			Expect(parsedTestResults.Summary.Failed).To(Equal(1))
			Expect(parsedTestResults.Summary.Canceled).To(Equal(1))
			Expect(parsedTestResults.Summary.TimedOut).To(Equal(1))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("bin/your-script {{ jsonFilePath }}")
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
					{ID: &id3, Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)}},
					{ID: &id4, Attempt: v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)}},
					{ID: &id5, Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
					{ID: &id6, Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
				},
			}

			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"jsonFilePath": "json-temp-file",
					},
				},
			))

			parsedTestResults := v1.TestResults{}
			decoder := json.NewDecoder(strings.NewReader(mockFile.Builder.String()))
			err = decoder.Decode(&parsedTestResults)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedTestResults.Summary.Tests).To(Equal(1))
			Expect(parsedTestResults.Summary.Failed).To(Equal(1))
			Expect(parsedTestResults.Summary.Canceled).To(Equal(0))
			Expect(parsedTestResults.Summary.TimedOut).To(Equal(0))
		})
	})

	Describe("CleanUp", func() {
		It("errs with more than one set of substitutions", func() {
			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			err := substitution.CleanUp([]map[string]string{{}, {}})
			Expect(err).To(HaveOccurred())
		})

		It("returns nil with no sets of substitutions", func() {
			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			err := substitution.CleanUp([]map[string]string{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("errs when the substitution doesn't have the file path", func() {
			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			err := substitution.CleanUp([]map[string]string{{"not-the-path": ""}})
			Expect(err).To(HaveOccurred())
		})

		It("errs when the file at the path in the substitution can't be removed", func() {
			mockFileSystem.MockRemove = func(_ string) error {
				return errors.NewInternalError("uh oh")
			}

			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			err := substitution.CleanUp([]map[string]string{{"jsonFilePath": "the-path"}})
			Expect(err).To(HaveOccurred())
		})

		It("removes the file at the path in the substitution", func() {
			calledWith := ""
			mockFileSystem.MockRemove = func(name string) error {
				calledWith = name
				return nil
			}

			substitution := targetedretries.JSONSubstitution{FileSystem: mockFileSystem}
			err := substitution.CleanUp([]map[string]string{{"jsonFilePath": "the-path"}})
			Expect(err).NotTo(HaveOccurred())
			Expect(calledWith).To(Equal("the-path"))
		})
	})
})
