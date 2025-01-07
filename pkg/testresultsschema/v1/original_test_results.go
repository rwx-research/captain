package testresultsschema

type OriginalTestResults struct {
	OriginalFilePath string `json:"originalFilePath"`
	Contents         string `json:"contents"`
	GroupNumber      int    `json:"groupNumber"`
}
