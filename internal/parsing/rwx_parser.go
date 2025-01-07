package parsing

import (
	"encoding/json"
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

type RWXParser struct{}

func (p RWXParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var testResults v1.TestResults

	if err := json.NewDecoder(data).Decode(&testResults); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}

	return &testResults, nil
}
