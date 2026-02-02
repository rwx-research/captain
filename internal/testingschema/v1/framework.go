package v1

import (
	"fmt"
	"strings"
)

type FrameworkLanguage string

type FrameworkKind string

const (
	FrameworkKindCucumber   FrameworkKind = "Cucumber"
	FrameworkKindCypress    FrameworkKind = "Cypress"
	FrameworkKindExUnit     FrameworkKind = "ExUnit"
	FrameworkKindGinkgo     FrameworkKind = "Ginkgo"
	FrameworkKindGoTest     FrameworkKind = "go test"
	FrameworkKindJest       FrameworkKind = "Jest"
	FrameworkKindKarma      FrameworkKind = "Karma"
	FrameworkKindMinitest   FrameworkKind = "minitest"
	FrameworkKindMocha      FrameworkKind = "Mocha"
	FrameworkKindPHPUnit    FrameworkKind = "PHPUnit"
	FrameworkKindPlaywright FrameworkKind = "Playwright"
	FrameworkKindPytest     FrameworkKind = "pytest"
	FrameworkKindUnitTest   FrameworkKind = "unittest"
	FrameworkKindRSpec      FrameworkKind = "RSpec"
	FrameworkKindxUnit      FrameworkKind = "xUnit"
	FrameworkKindVitest     FrameworkKind = "Vitest"
	FrameworkKindBun        FrameworkKind = "Bun"

	FrameworkLanguageDotNet     FrameworkLanguage = ".NET"
	FrameworkLanguageElixir     FrameworkLanguage = "Elixir"
	FrameworkLanguageGo         FrameworkLanguage = "Go"
	FrameworkLanguageJavaScript FrameworkLanguage = "JavaScript"
	FrameworkLanguagePHP        FrameworkLanguage = "PHP"
	FrameworkLanguagePython     FrameworkLanguage = "Python"
	FrameworkLanguageRuby       FrameworkLanguage = "Ruby"

	FrameworkKindOther     FrameworkKind     = "other"
	FrameworkLanguageOther FrameworkLanguage = "other"
)

type Framework struct {
	Language         FrameworkLanguage `json:"language"`
	Kind             FrameworkKind     `json:"kind"`
	ProvidedLanguage *string           `json:"providedLanguage,omitempty"`
	ProvidedKind     *string           `json:"providedKind,omitempty"`
}

var KnownFrameworks []Framework

func registerFramework(framework Framework) Framework {
	KnownFrameworks = append(KnownFrameworks, framework)
	return framework
}

var (
	DotNetxUnitFramework = registerFramework(
		Framework{Language: FrameworkLanguageDotNet, Kind: FrameworkKindxUnit},
	)
	ElixirExUnitFramework = registerFramework(
		Framework{Language: FrameworkLanguageElixir, Kind: FrameworkKindExUnit},
	)
	GoGinkgoFramework = registerFramework(
		Framework{Language: FrameworkLanguageGo, Kind: FrameworkKindGinkgo},
	)
	GoTestFramework = registerFramework(
		Framework{Language: FrameworkLanguageGo, Kind: FrameworkKindGoTest},
	)
	JavaScriptCucumberFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindCucumber},
	)
	JavaScriptCypressFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindCypress},
	)
	JavaScriptJestFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindJest},
	)
	JavaScriptKarmaFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindKarma},
	)
	JavaScriptMochaFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindMocha},
	)
	JavaScriptPlaywrightFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindPlaywright},
	)
	JavaScriptVitestFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindVitest},
	)
	JavaScriptBunFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindBun},
	)
	PHPUnitFramework = registerFramework(
		Framework{Language: FrameworkLanguagePHP, Kind: FrameworkKindPHPUnit},
	)
	PythonPytestFramework = registerFramework(
		Framework{Language: FrameworkLanguagePython, Kind: FrameworkKindPytest},
	)
	PythonUnitTestFramework = registerFramework(
		Framework{Language: FrameworkLanguagePython, Kind: FrameworkKindUnitTest},
	)
	RubyCucumberFramework = registerFramework(
		Framework{Language: FrameworkLanguageRuby, Kind: FrameworkKindCucumber},
	)
	RubyMinitestFramework = registerFramework(
		Framework{Language: FrameworkLanguageRuby, Kind: FrameworkKindMinitest},
	)
	RubyRSpecFramework = registerFramework(
		Framework{Language: FrameworkLanguageRuby, Kind: FrameworkKindRSpec},
	)
)

func NewOtherFramework(providedLanguage *string, providedKind *string) Framework {
	return Framework{
		Language:         FrameworkLanguageOther,
		Kind:             FrameworkKindOther,
		ProvidedLanguage: providedLanguage,
		ProvidedKind:     providedKind,
	}
}

func CoerceFramework(providedLanguage string, providedKind string) Framework {
	framework := NewOtherFramework(&providedLanguage, &providedKind)

	for _, knownFramework := range KnownFrameworks {
		if !strings.EqualFold(string(knownFramework.Language), strings.TrimSpace(providedLanguage)) {
			continue
		}

		if !strings.EqualFold(string(knownFramework.Kind), strings.TrimSpace(providedKind)) {
			continue
		}

		framework = knownFramework
		break
	}

	return framework
}

func (f Framework) Equal(f2 Framework) bool {
	if f.IsOther() && f2.IsOther() {
		if f.IsProvided() && f2.IsProvided() {
			return *f.ProvidedLanguage == *f2.ProvidedLanguage && *f.ProvidedKind == *f2.ProvidedKind
		}

		return !f.IsProvided() && !f2.IsProvided()
	}

	return !f.IsOther() && !f2.IsOther() && f.Kind == f2.Kind && f.Language == f2.Language
}

func (f Framework) IsOther() bool {
	return f.Language == FrameworkLanguageOther && f.Kind == FrameworkKindOther
}

func (f Framework) IsProvided() bool {
	return f.ProvidedLanguage != nil && f.ProvidedKind != nil
}

func (f Framework) String() string {
	if f.IsOther() {
		if f.IsProvided() {
			return fmt.Sprintf("Other: %v (%v)", *f.ProvidedKind, *f.ProvidedLanguage)
		}

		return "Other"
	}

	return fmt.Sprintf("%v (%v)", f.Kind, f.Language)
}
