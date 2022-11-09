package cli

import (
	"context"
	"io"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// APIClient is the interface of our API layer.
type APIClient interface {
	GetQuarantinedTestIDs(context.Context) ([]string, error)
	UploadArtifacts(context.Context, string, []api.Artifact) error
}

// FileSystem is an abstraction over file-systems. This is implemented by the default `os` package and can also be used
// for mocking.
type FileSystem interface {
	Open(name string) (fs.File, error)
}

// Parser is the interface a potential parser needs to implement.
type Parser interface {
	Parse(io.Reader) (map[string]testing.TestResult, error)
}

// TaskRunner is an abstraction over various task-runners / execution environments.
// They are expected to implement the `taskRunner.Command` interface in turn, which is mapped to the Command type from
// `os/exec`
type TaskRunner interface {
	NewCommand(ctx context.Context, name string, args ...string) (exec.Command, error)
	GetExitStatusFromError(error) (int, error)
}
