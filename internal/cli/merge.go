package cli

import (
	"context"
	"os"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
	"github.com/rwx-research/captain-cli/internal/reporting"
)

// Parse parses the files supplied in `filepaths` and reports them as specified
func (s Service) Merge(_ context.Context, cfg MergeConfig) error {
	testResultsFiles := make([]string, 0)
	for _, resultsGlob := range cfg.ResultsGlobs {
		files, err := s.FileSystem.Glob(resultsGlob)
		if err != nil {
			return errors.NewSystemError("unable to expand %q: %s", resultsGlob, err)
		}

		testResultsFiles = append(testResultsFiles, files...)
	}

	results, err := s.parse(testResultsFiles, 1)
	if err != nil {
		return errors.WithStack(err)
	}

	reportingConfiguration := reporting.Configuration{
		CloudEnabled:          false,
		CloudHost:             "",
		CloudOrganizationSlug: "",
		Provider:              providers.Provider{},
	}

	for outputPath, writeReport := range cfg.Reporters {
		file, err := s.FileSystem.Create(outputPath)
		if err == nil {
			err = writeReport(file, *results, reportingConfiguration)
		}
		if err != nil {
			s.Log.Warnf("Unable to write report to %s: %s", outputPath, err.Error())
		}
	}

	if cfg.PrintSummary {
		if err := reporting.WriteTextSummary(os.Stdout, *results, reportingConfiguration); err != nil {
			s.Log.Warnf("Unable to write text summary to stdout: %s", err.Error())
		}
	}

	return nil
}
