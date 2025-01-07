package testresultsschema_test

import (
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Framework", func() {
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
			Expect(v1.RubyRSpecFramework.IsOther()).To(BeFalse())
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
			Expect(v1.RubyRSpecFramework.IsProvided()).To(BeFalse())
		})
	})

	Describe("CoerceFramework", func() {
		It("can construct a well-known framework", func() {
			providedLanguage := "rUbY"
			providedKind := "rspec"
			framework := v1.CoerceFramework(providedLanguage, providedKind)
			Expect(framework).To(Equal(v1.RubyRSpecFramework))
		})

		It("cannot mix well-known languages and kinds", func() {
			providedLanguage := "rUbY"
			providedKind := "mocha"
			framework := v1.CoerceFramework(providedLanguage, providedKind)
			Expect(framework).To(Equal(v1.NewOtherFramework(&providedLanguage, &providedKind)))
		})

		It("returns other frameworks with their provided kind and language", func() {
			providedLanguage := "something"
			providedKind := "unknown"
			framework := v1.CoerceFramework(providedLanguage, providedKind)
			Expect(framework).To(Equal(v1.NewOtherFramework(&providedLanguage, &providedKind)))
		})
	})

	Describe("String", func() {
		It("has a good string representation for provided other frameworks", func() {
			providedLanguage := "provided language"
			providedKind := "provided kind"
			Expect(v1.NewOtherFramework(&providedLanguage, &providedKind).String()).To(
				Equal("Other: provided kind (provided language)"),
			)
		})

		It("has a good string representation for unprovided other frameworks", func() {
			Expect(v1.NewOtherFramework(nil, nil).String()).To(
				Equal("Other"),
			)
		})

		It("has a good string representation for known frameworks", func() {
			Expect(v1.RubyRSpecFramework.String()).To(
				Equal("RSpec (Ruby)"),
			)
		})
	})
})
