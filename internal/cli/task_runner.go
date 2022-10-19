package cli

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/exec"
)

// TaskRunner is an abstraction over various task-runners / execution environments.
// They are expected to implement the `taskRunner.Command` interface in turn, which is mapped to the Command type from
// `os/exec`
type TaskRunner interface {
	NewCommand(ctx context.Context, name string, args ...string) (exec.Command, error)
	GetExitStatusFromError(error) (int, error)
}
