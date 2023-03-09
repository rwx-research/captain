// Package cli holds the main business logic in our CLI. This is mainly:
// 1. Triggering the right API calls based on the provided input parameters.
// 2. User-friendly logging
// However, this package _does not_ implement the actual terminal UI. That part is handled by `cmd/captain`.
package cli

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// Service is the main CLI service.
type Service struct {
	API         APIClient
	Log         *zap.SugaredLogger
	FileSystem  fs.FileSystem
	TaskRunner  TaskRunner
	ParseConfig parsing.Config
}

func (s Service) logError(err error) error {
	s.Log.Errorf(err.Error())
	return err
}

// Parse parses the files supplied in `filepaths` and prints them as formatted JSON to stdout.
func (s Service) Parse(ctx context.Context, filepaths []string) error {
	results, err := s.parse(filepaths, 1)
	if err != nil {
		return s.logError(errors.WithStack(err))
	}

	newOutput, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return s.logError(errors.NewInternalError("Unable to output test results as JSON: %s", err))
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

// UploadTestResults is the implementation of `captain upload results`. It takes a number of filepaths, checks that
// files exist and uploads them using the provided API.
func (s Service) UploadTestResults(
	ctx context.Context,
	testSuiteID string,
	filepaths []string,
) ([]api.TestResultsUploadResult, error) {
	if testSuiteID == "" {
		return nil, errors.NewConfigurationError("suite-id is required")
	}

	if len(filepaths) == 0 {
		s.Log.Debug("No paths to test results provided")
		return nil, nil
	}

	expandedFilepaths, err := s.FileSystem.GlobMany(filepaths)
	if err != nil {
		return nil, s.logError(errors.NewSystemError("unable to expand filepath glob: %s", err))
	}
	if len(expandedFilepaths) == 0 {
		s.Log.Debug("No paths to test results provided")
		return nil, nil
	}

	parsedResults, err := s.parse(expandedFilepaths, 1)
	if err != nil {
		return nil, s.logError(errors.WithStack(err))
	}

	return s.performTestResultsUpload(
		ctx,
		testSuiteID,
		[]v1.TestResults{*parsedResults},
	)
}

func (s Service) performTestResultsUpload(
	ctx context.Context,
	testSuiteID string,
	testResults []v1.TestResults,
) ([]api.TestResultsUploadResult, error) {
	testResultsFiles := make([]api.TestResultsFile, len(testResults))

	for i, testResult := range testResults {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, s.logError(errors.NewInternalError("Unable to generate new UUID: %s", err))
		}

		buf, err := json.Marshal(testResult)
		if err != nil {
			return nil, s.logError(errors.NewInternalError("Unable to output test results as JSON: %s", err))
		}

		f := fs.VirtualReadOnlyFile{
			Reader:   bytes.NewReader(buf),
			FileName: "rwx-test-results",
		}

		originalPaths := make([]string, len(testResult.DerivedFrom))
		for i, originalTestResult := range testResult.DerivedFrom {
			originalPaths[i] = originalTestResult.OriginalFilePath
		}

		testResultsFiles[i] = api.TestResultsFile{
			ExternalID:    id,
			FD:            f,
			OriginalPaths: originalPaths,
			Parser:        api.ParserTypeRWX,
		}
	}

	uploadResults, err := s.API.UploadTestResults(ctx, testSuiteID, testResultsFiles)
	if err != nil {
		return nil, s.logError(errors.Wrap(err, "unable to upload test result"))
	}

	return uploadResults, nil
}
