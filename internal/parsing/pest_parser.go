package parsing

import (
	"io"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type PestParser struct{}

func (p PestParser) Parse(data io.Reader) (*v1.TestResults, error) {
	results, err := (PHPUnitParser{}).Parse(data)
	if err != nil {
		return nil, err
	}
	results.Framework = v1.PHPPestFramework
	return results, nil
}
