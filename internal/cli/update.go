package cli

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/backend/local"
	"github.com/rwx-research/captain-cli/internal/errors"
)

// UploadTestResults is the implementation of `captain upload results`.
// Deprecated: Use `captain update results` instead, which supports both the local and remote backend.
func (s Service) UploadTestResults(
	ctx context.Context,
	testSuiteID string,
	filepaths []string,
) ([]backend.TestResultsUploadResult, error) {
	// only attempt the upload if the CLI is set up to interact with Captain Cloud.
	if _, ok := s.API.(local.Client); ok {
		return nil, errors.NewConfigurationError("Uploading test results requires a Captain Cloud subscription.")
	}

	return s.UpdateTestResults(ctx, testSuiteID, filepaths)
}

// UpdateTestResults is the implementation of `captain update results`. It takes a number of filepaths, checks that
// files exist and uploads them using the provided API.
func (s Service) UpdateTestResults(
	ctx context.Context,
	testSuiteID string,
	filepaths []string,
) ([]backend.TestResultsUploadResult, error) {
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

	result, err := s.API.UpdateTestResults(ctx, testSuiteID, *parsedResults)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update test results")
	}

	return result, nil
}
