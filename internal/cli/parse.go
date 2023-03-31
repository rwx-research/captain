package cli

import (
	"context"
	"encoding/json"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// Parse parses the files supplied in `filepaths` and prints them as formatted JSON to stdout.
func (s Service) Parse(_ context.Context, filepaths []string) error {
	results, err := s.parse(filepaths, 1)
	if err != nil {
		return errors.WithStack(err)
	}

	newOutput, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return errors.NewInternalError("Unable to output test results as JSON: %s", err)
	}
	s.Log.Infoln(string(newOutput))

	return nil
}

func (s Service) parse(filepaths []string, group int) (*v1.TestResults, error) {
	var framework *v1.Framework
	allResults := make([]v1.TestResults, 0)

	for _, testResultsFilePath := range filepaths {
		s.Log.Debugf("Attempting to parse %q", testResultsFilePath)

		fd, err := s.FileSystem.Open(testResultsFilePath)
		if err != nil {
			return nil, errors.NewSystemError("unable to open file: %s", err)
		}
		defer fd.Close()

		results, err := parsing.Parse(fd, group, s.ParseConfig)
		if err != nil {
			return nil, errors.NewInputError("Unable to parse %q with the available parsers", testResultsFilePath)
		}

		if framework == nil {
			framework = &results.Framework
		} else if *framework != results.Framework {
			return nil, errors.NewInputError(
				"Multiple frameworks detected. The captain CLI only works with one framework at a time",
			)
		}

		allResults = append(allResults, *results)
	}

	if len(allResults) == 0 {
		return nil, nil
	}

	mergedResults := v1.Merge(allResults)
	return &mergedResults, nil
}
