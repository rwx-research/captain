package v1

import "strings"

type FrameworkLanguage string

type FrameworkKind string

const (
	FrameworkKindCucumber FrameworkKind = "Cucumber"
	FrameworkKindCypress  FrameworkKind = "Cypress"
	FrameworkKindJest     FrameworkKind = "Jest"
	FrameworkKindMocha    FrameworkKind = "Mocha"
	FrameworkKindPytest   FrameworkKind = "pytest"
	FrameworkKindRSpec    FrameworkKind = "RSpec"
	FrameworkKindxUnit    FrameworkKind = "xUnit"

	FrameworkLanguageDotNet     FrameworkLanguage = ".NET"
	FrameworkLanguageJavaScript FrameworkLanguage = "JavaScript"
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
	JavaScriptCypressFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindCypress},
	)
	JavaScriptJestFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindJest},
	)
	JavaScriptMochaFramework = registerFramework(
		Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindMocha},
	)
	PythonPytestFramework = registerFramework(
		Framework{Language: FrameworkLanguagePython, Kind: FrameworkKindPytest},
	)
	RubyRSpecFramework = registerFramework(
		Framework{Language: FrameworkLanguageRuby, Kind: FrameworkKindRSpec},
	)
	RubyCucumberFramework = registerFramework(
		Framework{Language: FrameworkLanguageRuby, Kind: FrameworkKindCucumber},
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
