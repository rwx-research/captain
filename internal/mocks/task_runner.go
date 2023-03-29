package mocks

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
)

// TaskRunner is a mocked implementation of 'cli.TaskRunner'.
type TaskRunner struct {
	MockNewCommand             func(ctx context.Context, cfg exec.CommandConfig) (exec.Command, error)
	MockGetExitStatusFromError func(error) (int, error)
}

// NewCommand either calls the configured mock of itself or returns an error if that doesn't exist.
func (t *TaskRunner) NewCommand(ctx context.Context, cfg exec.CommandConfig) (exec.Command, error) {
	if t.MockNewCommand != nil {
		return t.MockNewCommand(ctx, cfg)
	}

	return nil, errors.NewInternalError("MockNewCommand was not configured")
}

// GetExitStatusFromError either calls the configured mock of itself or returns an error if that doesn't exist.
func (t *TaskRunner) GetExitStatusFromError(err error) (int, error) {
	if t.MockGetExitStatusFromError != nil {
		return t.MockGetExitStatusFromError(err)
	}

	return 0, errors.NewInternalError("MockGetExitStatusFromError was not configured")
}
