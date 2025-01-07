package reporting

import (
	"encoding/json"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

func WriteJSONSummary(file fs.File, testResults v1.TestResults, _ Configuration) error {
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(testResults); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
