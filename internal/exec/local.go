// Package exec exposes various task runners that can execute arbitrary commands. This is mostly a thin wrapper around
// `os/exec` plus a mocked implementation. We could extend this in the future to support for remote task runners (abq or
// ssh come to mind).
package exec

import (
	"context"
	"os/exec"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// Local is a local executioner. It wraps `os/exec`
type Local struct{}

// NewCommand returns a new command that can then be executed.
func (l Local) NewCommand(ctx context.Context, cfg CommandConfig) (Command, error) {
	//nolint:gosec // Spawning a user-configurable sub-process is expected here.
	cmd := exec.CommandContext(ctx, cfg.Name, cfg.Args...)

	cmd.Stderr = cfg.Stderr
	cmd.Stdout = cfg.Stdout

	for _, override := range cfg.Env {
		cmd.Env = append(cmd.Environ(), override)
	}

	return cmd, nil
}

// GetExitStatus extracts the exit code from an error
func (l Local) GetExitStatusFromError(err error) (int, error) {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode(), nil
	}

	return 0, errors.NewInternalError("Expected error to be of type exec.ExitError, received %T", err)
}
