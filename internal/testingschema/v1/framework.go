package v1

type FrameworkLanguage string

type FrameworkKind string

const (
	FrameworkLanguageRuby       FrameworkLanguage = "Ruby"
	FrameworkLanguageJavaScript FrameworkLanguage = "JavaScript"
	FrameworkLanguageDotNet     FrameworkLanguage = ".NET"
	FrameworkLanguageOther      FrameworkLanguage = "other"
	FrameworkKindRSpec          FrameworkKind     = "RSpec"
	FrameworkKindxUnit          FrameworkKind     = "xUnit"
	FrameworkKindJest           FrameworkKind     = "Jest"
	FrameworkKindOther          FrameworkKind     = "other"
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

func NewJavaScriptJestFramework() Framework {
	return Framework{Language: FrameworkLanguageJavaScript, Kind: FrameworkKindJest}
}

func NewDotNetxUnitFramework() Framework {
	return Framework{Language: FrameworkLanguageDotNet, Kind: FrameworkKindxUnit}
}

func NewOtherFramework(providedLanguage string, providedKind string) Framework {
	return Framework{
		Language:         FrameworkLanguageOther,
		Kind:             FrameworkKindOther,
		ProvidedLanguage: &providedLanguage,
		ProvidedKind:     &providedKind,
	}
}
