package mocks

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"
)

// API is a mocked implementation of 'api.Client'.
type API struct {
	MockGetQuarantinedTestCases func(context.Context, string) ([]api.QuarantinedTestCase, error)
	MockUploadTestResults       func(context.Context, string, []api.TestResultsFile) ([]api.TestResultsUploadResult, error)
}

// GetQuarantinedTestCases either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) GetQuarantinedTestCases(
	ctx context.Context,
	testSuiteIdentifier string,
) ([]api.QuarantinedTestCase, error) {
	if a.MockGetQuarantinedTestCases != nil {
		return a.MockGetQuarantinedTestCases(ctx, testSuiteIdentifier)
	}

	return nil, errors.NewConfigurationError("MockGetQuarantinedTestCases was not configured")
}

// UploadTestResults either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) UploadTestResults(
	ctx context.Context,
	testSuiteID string,
	testResultsFiles []api.TestResultsFile,
) ([]api.TestResultsUploadResult, error) {
	if a.MockUploadTestResults != nil {
		return a.MockUploadTestResults(ctx, testSuiteID, testResultsFiles)
	}

	return nil, errors.NewConfigurationError("MockUploadTestResults was not configured")
}
