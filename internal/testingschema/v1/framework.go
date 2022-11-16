package v1

type Framework struct {
	Language         string  `json:"language"`
	Kind             string  `json:"kind"`
	ProvidedLanguage *string `json:"providedLanguage,omitempty"`
	ProvidedKind     *string `json:"providedKind,omitempty"`
}
