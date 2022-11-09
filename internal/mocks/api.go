package mocks

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"
)

// API is a mocked implementation of 'api.Client'.
type API struct {
	MockGetQuarantinedTestIDs func(context.Context) ([]string, error)
	MockUploadArtifacts       func(context.Context, string, []api.Artifact) error
}

// GetQuarantinedTestIDs either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) GetQuarantinedTestIDs(ctx context.Context) ([]string, error) {
	if a.MockGetQuarantinedTestIDs != nil {
		return a.MockGetQuarantinedTestIDs(ctx)
	}

	return nil, errors.NewConfigurationError("MockGetQuarantinedTestIDs was not configured")
}

// UploadArtifacts either calls the configured mock of itself or returns an error if that doesn't exist.
func (a *API) UploadArtifacts(ctx context.Context, testSuite string, artifacts []api.Artifact) error {
	if a.MockUploadArtifacts != nil {
		return a.MockUploadArtifacts(ctx, testSuite, artifacts)
	}

	return errors.NewConfigurationError("MockUploadArtifacts was not configured")
}
