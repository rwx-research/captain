package v1

type OtherError struct {
	Backtrace []string       `json:"backtrace,omitempty"`
	Exception *string        `json:"exception,omitempty"`
	Location  *Location      `json:"location,omitempty"`
	Message   string         `json:"message"`
	Meta      map[string]any `json:"meta,omitempty"`
}
