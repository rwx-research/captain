package runpartition_test

import (
	"github.com/rwx-research/captain-cli/internal/runpartition"
	"github.com/rwx-research/captain-cli/internal/templating"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DelimiterSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution runpartition.Substitution = runpartition.DelimiterSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := runpartition.DelimiterSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := runpartition.DelimiterSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("nothing")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no keywords were found"))
		})

		It("is invalid for a template without multiple placeholders", func() {
			substitution := runpartition.DelimiterSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("some-command {{ a }} {{ b }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("these were found: a, b"))
		})

		It("is invalid for a template with one incorrect placeholders", func() {
			substitution := runpartition.DelimiterSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("some-command {{ testFilesNope }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'testFilesNope' was found instead"))
		})

		It("is valid exactly testFiles is provided", func() {
			substitution := runpartition.DelimiterSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("some-command {{ testFiles }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SubstitutionLookupFor", func() {
		Context("when provided no files", func() {
			It("returns testFiles as an empty string", func() {
				substitution := runpartition.DelimiterSubstitution{}
				compiledTemplate, _ := templating.CompileTemplate("some-command {{ testFiles }}")
				lookup, err := substitution.SubstitutionLookupFor(compiledTemplate, []string{})

				Expect(err).NotTo(HaveOccurred())
				Expect(lookup["testFiles"]).To(Equal(""))
			})
		})

		Context("when provided many files", func() {
			It("returns testFiles separated with delimiter", func() {
				substitution := runpartition.DelimiterSubstitution{Delimiter: " || "}
				compiledTemplate, _ := templating.CompileTemplate("some-command {{ testFiles }}")
				lookup, err := substitution.SubstitutionLookupFor(compiledTemplate, []string{"a", "b", "c"})

				Expect(err).NotTo(HaveOccurred())
				Expect(lookup["testFiles"]).To(Equal("'a' || 'b' || 'c'"))
				Expect(len(lookup)).To(Equal(1))
			})
		})

		Context("when file includes a space", func() {
			It("gets escaped", func() {
				substitution := runpartition.DelimiterSubstitution{Delimiter: ","}
				compiledTemplate, _ := templating.CompileTemplate("some-command {{ testFiles }}")
				lookup, err := substitution.SubstitutionLookupFor(compiledTemplate, []string{"a spec", "b spec"})

				Expect(err).NotTo(HaveOccurred())
				Expect(lookup["testFiles"]).To(Equal("'a spec','b spec'"))
			})
		})
	})
})
