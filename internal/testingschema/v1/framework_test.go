package v1_test

import (
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Framework", func() {
	Describe("NewRubyCucumberFramework", func() {
		It("produces a framework of the expected configuration", func() {
			Expect(v1.NewRubyCucumberFramework()).To(Equal(v1.Framework{
				Language: v1.FrameworkLanguageRuby,
				Kind:     v1.FrameworkKindCucumber,
			}))
		})
	})

	Describe("NewRubyRSpecFramework", func() {
		It("produces a framework of the expected configuration", func() {
			Expect(v1.NewRubyRSpecFramework()).To(Equal(v1.Framework{
				Language: v1.FrameworkLanguageRuby,
				Kind:     v1.FrameworkKindRSpec,
			}))
		})
	})

	Describe("NewJavaScriptCypressFramework", func() {
		It("produces a framework of the expected configuration", func() {
			Expect(v1.NewJavaScriptCypressFramework()).To(Equal(v1.Framework{
				Language: v1.FrameworkLanguageJavaScript,
				Kind:     v1.FrameworkKindCypress,
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

	Describe("NewJavaScriptMochaFramework", func() {
		It("produces a framework of the expected configuration", func() {
			Expect(v1.NewJavaScriptMochaFramework()).To(Equal(v1.Framework{
				Language: v1.FrameworkLanguageJavaScript,
				Kind:     v1.FrameworkKindMocha,
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
			Expect(v1.NewOtherFramework(&providedLanguage, &providedKind)).To(Equal(v1.Framework{
				Language:         v1.FrameworkLanguageOther,
				Kind:             v1.FrameworkKindOther,
				ProvidedLanguage: &providedLanguage,
				ProvidedKind:     &providedKind,
			}))
		})
	})

	Describe("IsOther", func() {
		It("is true when called for an other framework", func() {
			Expect(v1.NewOtherFramework(nil, nil).IsOther()).To(BeTrue())
		})

		It("is false when called for a known framework", func() {
			Expect(v1.NewRubyRSpecFramework().IsOther()).To(BeFalse())
		})
	})

	Describe("IsProvided", func() {
		It("is true when called for an other framework with all provided fields", func() {
			providedLanguage := "provided language"
			providedKind := "provided kind"
			Expect(v1.NewOtherFramework(&providedLanguage, &providedKind).IsProvided()).To(BeTrue())
		})

		It("is false when called for an other framework with some or no provided fields", func() {
			providedLanguage := "provided language"
			providedKind := "provided kind"
			Expect(v1.NewOtherFramework(&providedLanguage, nil).IsProvided()).To(BeFalse())
			Expect(v1.NewOtherFramework(nil, &providedKind).IsProvided()).To(BeFalse())
			Expect(v1.NewOtherFramework(nil, nil).IsProvided()).To(BeFalse())
		})

		It("is false when called for a known framework", func() {
			Expect(v1.NewRubyRSpecFramework().IsProvided()).To(BeFalse())
		})
	})
})
