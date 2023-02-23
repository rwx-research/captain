package targetedretries_test

import (
	"github.com/rwx-research/captain-cli/internal/targetedretries"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var substitutions = map[string]string{
	"keyword1": "one",
	"keyword2": "two",
	"keyword3": "three",
	"keyword4": "four",
}

var _ = Describe("CompileTemplate", func() {
	It("errs when the same placeholder is specified multiple times", func() {
		template := "some-command '{{ keyword1 }}' '{{ keyword1 }}'"
		_, err := targetedretries.CompileTemplate(template)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"template requested duplicate substitution of placeholder 'keyword1'",
		))
	})

	It("compiles with no mapping when there are no placeholders", func() {
		template := "some-command"
		compiledTemplate, err := targetedretries.CompileTemplate(template)

		Expect(err).NotTo(HaveOccurred())
		Expect(compiledTemplate.Keywords()).To(Equal([]string{}))
		Expect(compiledTemplate.PlaceholderToKeyword).To(BeEmpty())
		Expect(compiledTemplate.Template).To(Equal(template))
	})

	It("compiles one placeholder", func() {
		template := "some-command '{{keyword1}}}'"
		compiledTemplate, err := targetedretries.CompileTemplate(template)

		Expect(err).NotTo(HaveOccurred())
		Expect(compiledTemplate.Keywords()).To(ConsistOf("keyword1"))
		Expect(compiledTemplate.PlaceholderToKeyword).To(Equal(map[string]string{"{{keyword1}}": "keyword1"}))
		Expect(compiledTemplate.Template).To(Equal(template))
	})

	It("compiles multiple placeholder", func() {
		template := "some-command '{{keyword1}}' '{{ keyword2 }}' '{{keyword3 }}' '{{ keyword4}}'"
		compiledTemplate, err := targetedretries.CompileTemplate(template)

		Expect(err).NotTo(HaveOccurred())
		Expect(compiledTemplate.Keywords()).To(ConsistOf("keyword1", "keyword2", "keyword3", "keyword4"))
		Expect(compiledTemplate.PlaceholderToKeyword).To(Equal(
			map[string]string{
				"{{keyword1}}":   "keyword1",
				"{{ keyword2 }}": "keyword2",
				"{{keyword3 }}":  "keyword3",
				"{{ keyword4}}":  "keyword4",
			},
		))
		Expect(compiledTemplate.Template).To(Equal(template))
	})
})

var _ = Describe("Substitute", func() {
	It("returns the template when no substitutions are needed", func() {
		compiledTemplate := targetedretries.CompiledTemplate{
			Template:             "some-command",
			PlaceholderToKeyword: map[string]string{},
		}
		substituted := compiledTemplate.Substitute(substitutions)

		Expect(substituted).To(Equal("some-command"))
	})

	It("can substitute just once", func() {
		compiledTemplate := targetedretries.CompiledTemplate{
			Template: "some-command '{{ keyword1 }}'",
			PlaceholderToKeyword: map[string]string{
				"{{ keyword1 }}": "keyword1",
			},
		}
		substituted := compiledTemplate.Substitute(substitutions)

		Expect(substituted).To(Equal("some-command 'one'"))
	})

	It("can substitute multiple times", func() {
		compiledTemplate := targetedretries.CompiledTemplate{
			Template: "some-command '{{ keyword1 }}' '{{keyword2}}'",
			PlaceholderToKeyword: map[string]string{
				"{{ keyword1 }}": "keyword1",
				"{{keyword2}}":   "keyword2",
			},
		}
		substituted := compiledTemplate.Substitute(substitutions)

		Expect(substituted).To(Equal("some-command 'one' 'two'"))
	})
})
