package v1

import "strings"

type FrameworkLanguage string

type FrameworkKind string

const (
	FrameworkKindCucumber   FrameworkKind = "Cucumber"
	FrameworkKindCypress    FrameworkKind = "Cypress"
	FrameworkKindGinkgo     FrameworkKind = "Ginkgo"
	FrameworkKindGoTest     FrameworkKind = "go test"
	FrameworkKindJest       FrameworkKind = "Jest"
	FrameworkKindMinitest   FrameworkKind = "minitest"
	FrameworkKindMocha      FrameworkKind = "Mocha"
	FrameworkKindPHPUnit    FrameworkKind = "PHPUnit"
	FrameworkKindPlaywright FrameworkKind = "Playwright"
	FrameworkKindPytest     FrameworkKind = "pytest"
	FrameworkKindRSpec      FrameworkKind = "RSpec"
	FrameworkKindxUnit      FrameworkKind = "xUnit"

	FrameworkLanguageDotNet     FrameworkLanguage = ".NET"
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
	GoGinkgoFramework = registerFramework(
		Framework{Language: FrameworkLanguageGo, Kind: FrameworkKindGinkgo},
	)
	GoTestFramework = registerFramework(
		Framework{Language: FrameworkLanguageGo, Kind: FrameworkKindGoTest},
	)
	JavaScriptCypressFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindCypress},
	)
	JavaScriptJestFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindJest},
	)
	JavaScriptMochaFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindMocha},
	)
	JavaScriptPlaywrightFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindPlaywright},
	)
	PHPUnitFramework = registerFramework(
		Framework{Language: FrameworkLanguagePHP, Kind: FrameworkKindPHPUnit},
	)
	PythonPytestFramework = registerFramework(
		Framework{Language: FrameworkLanguagePython, Kind: FrameworkKindPytest},
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

func (f Framework) IsOther() bool {
	return f.Language == FrameworkLanguageOther && f.Kind == FrameworkKindOther
}

func (f Framework) IsProvided() bool {
	return f.ProvidedLanguage != nil && f.ProvidedKind != nil
}
