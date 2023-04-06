package cli

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/reporting"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// Reporter is a function that writes test results to a file. Different reporters implement different encodings.
type Reporter func(fs.File, v1.TestResults, reporting.Configuration) error

// TaskRunner is an abstraction over various task-runners / execution environments.
// They are expected to implement the `taskRunner.Command` interface in turn, which is mapped to the Command type from
// `os/exec`
type TaskRunner interface {
	NewCommand(ctx context.Context, cfg exec.CommandConfig) (exec.Command, error)
	GetExitStatusFromError(error) (int, error)
}
