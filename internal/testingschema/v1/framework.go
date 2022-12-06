package v1

type FrameworkLanguage string

type FrameworkKind string

const (
	FrameworkKindCypress  FrameworkKind = "Cypress"
	FrameworkKindCucumber FrameworkKind = "Cucumber"
	FrameworkKindJest     FrameworkKind = "Jest"
	FrameworkKindRSpec    FrameworkKind = "RSpec"
	FrameworkKindxUnit    FrameworkKind = "xUnit"

	FrameworkLanguageDotNet     FrameworkLanguage = ".NET"
	FrameworkLanguageJavaScript FrameworkLanguage = "JavaScript"
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

func NewRubyRSpecFramework() Framework {
	return Framework{Language: FrameworkLanguageRuby, Kind: FrameworkKindRSpec}
}

func NewRubyCucumberFramework() Framework {
	return Framework{Language: FrameworkLanguageRuby, Kind: FrameworkKindCucumber}
}

func NewJavaScriptJestFramework() Framework {
	return Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindJest}
}

func NewJavaScriptCypressFramework() Framework {
	return Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindCypress}
}

func NewDotNetxUnitFramework() Framework {
	return Framework{Language: FrameworkLanguageDotNet, Kind: FrameworkKindxUnit}
}

func NewOtherFramework(providedLanguage *string, providedKind *string) Framework {
	return Framework{
		Language:         FrameworkLanguageOther,
		Kind:             FrameworkKindOther,
		ProvidedLanguage: providedLanguage,
		ProvidedKind:     providedKind,
	}
}

func (f Framework) IsOther() bool {
	return f.Language == FrameworkLanguageOther && f.Kind == FrameworkKindOther
}

func (f Framework) IsProvided() bool {
	return f.ProvidedLanguage != nil && f.ProvidedKind != nil
}
