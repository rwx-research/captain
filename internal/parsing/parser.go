package parsing

import (
	"io"

	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

type Parser interface {
	Parse(io.Reader) (*v1.TestResults, error)
}
