package v1_test

import (
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Framework", func() {
	Describe("NewRubyRSpecFramework", func() {
		It("produces a framework of the expected configuration", func() {
			Expect(v1.NewRubyRSpecFramework()).To(Equal(v1.Framework{
				Language: v1.FrameworkLanguageRuby,
				Kind:     v1.FrameworkKindRSpec,
			}))
		})
	})

	Describe("NewJavaScriptJestFramework", func() {
		It("produces a framework of the expected configuration", func() {
			Expect(v1.NewJavaScriptJestFramework()).To(Equal(v1.Framework{
				Language: v1.FrameworkLanguageJavaScript,
				Kind:     v1.FrameworkKindJest,
			}))
		})
	})

	Describe("NewDotNetxUnitFramework", func() {
		It("produces a framework of the expected configuration", func() {
			Expect(v1.NewDotNetxUnitFramework()).To(Equal(v1.Framework{
				Language: v1.FrameworkLanguageDotNet,
				Kind:     v1.FrameworkKindxUnit,
			}))
		})
	})

	Describe("NewOtherFramework", func() {
		It("produces a framework of the expected configuration", func() {
			providedLanguage := "provided language"
			providedKind := "provided kind"
			Expect(v1.NewOtherFramework(providedLanguage, providedKind)).To(Equal(v1.Framework{
				Language:         v1.FrameworkLanguageOther,
				Kind:             v1.FrameworkKindOther,
				ProvidedLanguage: &providedLanguage,
				ProvidedKind:     &providedKind,
			}))
		})
	})
})
