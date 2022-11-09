package mocks

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"
)

// API is a mocked implementation of 'api.Client'.
type API struct {
	MockGetQuarantinedTestIDs func(context.Context) ([]string, error)
	MockUploadTestResults     func(context.Context, string, []api.TestResultsFile) error
}

// GetQuarantinedTestIDs either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) GetQuarantinedTestIDs(ctx context.Context) ([]string, error) {
	if a.MockGetQuarantinedTestIDs != nil {
		return a.MockGetQuarantinedTestIDs(ctx)
	}

	return nil, errors.NewConfigurationError("MockGetQuarantinedTestIDs was not configured")
}

// UploadTestResults either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) UploadTestResults(ctx context.Context, testSuite string, testResultsFiles []api.TestResultsFile) error {
	if a.MockUploadTestResults != nil {
		return a.MockUploadTestResults(ctx, testSuite, testResultsFiles)
	}

	return errors.NewConfigurationError("MockUploadTestResults was not configured")
}
