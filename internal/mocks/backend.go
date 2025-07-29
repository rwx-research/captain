package mocks

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/testing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// API is a mocked implementation of 'backend.Client'.
type API struct {
	MockGetRunConfiguration   func(context.Context, string) (backend.RunConfiguration, error)
	MockGetQuarantinedTests   func(context.Context, string) ([]backend.Test, error)
	MockGetTestTimingManifest func(context.Context, string) ([]testing.TestFileTiming, error)
	MockUpdateTestResults     func(context.Context, string, v1.TestResults) (
		[]backend.TestResultsUploadResult, error,
	)
}

// GetRunConfiguration either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) GetRunConfiguration(
	ctx context.Context,
	testSuiteIdentifier string,
) (backend.RunConfiguration, error) {
	if a.MockGetRunConfiguration != nil {
		return a.MockGetRunConfiguration(ctx, testSuiteIdentifier)
	}

	return backend.RunConfiguration{}, errors.NewInternalError("MockGetRunConfiguration was not configured")
}

// GetQuarantinedTests either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) GetQuarantinedTests(
	ctx context.Context,
	testSuiteIdentifier string,
) ([]backend.Test, error) {
	if a.MockGetQuarantinedTests != nil {
		return a.MockGetQuarantinedTests(ctx, testSuiteIdentifier)
	}

	return nil, errors.NewInternalError("MockGetQuarantinedTests was not configured")
}

// GetTestTimingManifest either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) GetTestTimingManifest(
	ctx context.Context,
	testSuiteIdentifier string,
) ([]testing.TestFileTiming, error) {
	if a.MockGetTestTimingManifest != nil {
		return a.MockGetTestTimingManifest(ctx, testSuiteIdentifier)
	}

	return nil, errors.NewInternalError("MockGetTestTimingManifest was not configured")
}

// UploadTestResults either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) UpdateTestResults(
	ctx context.Context,
	testSuiteID string,
	testResults v1.TestResults,
) ([]backend.TestResultsUploadResult, error) {
	if a.MockUpdateTestResults != nil {
		return a.MockUpdateTestResults(ctx, testSuiteID, testResults)
	}

	return nil, errors.NewInternalError("MockUpdateTestResults was not configured")
}
