// parsing holds the functionality to attempt to parse a test results file and the parsers themselves
// the parsers will produce a testingschema test result or an error
// in the case where no parsers are capable of parsing test results, an error will be returned indicating so
package parsing

import (
	"encoding/base64"
	"io"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

func Parse(file fs.File, parsers []Parser, log *zap.SugaredLogger) (*v1.TestResults, error) {
	if len(parsers) == 0 {
		return nil, errors.NewInternalError("No parsers were provided")
	}
	if log == nil {
		return nil, errors.NewInternalError("No logger was provided")
	}

	parsedTestResults := make([]v1.TestResults, 0)
	var firstParser Parser
	for _, parser := range parsers {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, errors.NewSystemError("Unable to read from file: %s", err)
		}

		parsedTestResult, err := parser.Parse(file)
		if err != nil {
			log.Debugf("%T was not capable of parsing the test results. Error: %v", parser, err)
			continue
		}
		if parsedTestResult == nil {
			return nil, errors.NewInternalError("%T did not error and did not return a test result", parser)
		}
		log.Debugf("%T was capable of parsing the test results.", parser)

		if firstParser == nil {
			firstParser = parser
		}
		parsedTestResults = append(parsedTestResults, *parsedTestResult)
	}

	if len(parsedTestResults) == 0 {
		return nil, errors.NewInputError("No parsers were capable of parsing the provided test results")
	}

	finalResults := parsedTestResults[0]
	log.Debugf("%T was ultimately responsible for parsing the test results", firstParser)

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, errors.NewSystemError("Unable to read from file: %s", err)
	}
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.NewSystemError("Unable to read file into buffer: %s", err)
	}

	finalResults.DerivedFrom = []v1.OriginalTestResults{
		{
			OriginalFilePath: file.Name(),
			Contents:         base64.StdEncoding.EncodeToString(buf),
			GroupNumber:      1,
		},
	}

	return &finalResults, nil
}
