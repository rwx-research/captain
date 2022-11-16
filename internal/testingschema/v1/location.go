package v1

type Location struct {
	File   string `json:"file"`
	Line   *int   `json:"line,omitempty"`
	Column *int   `json:"column,omitempty"`
}
