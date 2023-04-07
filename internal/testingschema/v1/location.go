package v1

import "fmt"

type Location struct {
	File   string `json:"file"`
	Line   *int   `json:"line,omitempty"`
	Column *int   `json:"column,omitempty"`
}

func (l Location) String() string {
	if l.Line != nil {
		return fmt.Sprintf("%v:%v", l.File, *l.Line)
	}

	return l.File
}
