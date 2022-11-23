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
	GetQuarantinedTestCases(context.Context, string) ([]api.QuarantinedTestCase, error)
	GetTestTimingManifest(context.Context, string) ([]testing.TestFileTiming, error)
	UploadTestResults(context.Context, string, []api.TestResultsFile) ([]api.TestResultsUploadResult, error)
}

// FileSystem is an abstraction over file-systems. This is implemented by the default `os` package and can also be used
// for mocking.
type FileSystem interface {
	Open(name string) (fs.File, error)
	Glob(pattern string) ([]string, error)
	GlobMany(patterns []string) ([]string, error)
}

// Parser is the interface a potential parser needs to implement.
type Parser interface {
	Parse(io.Reader) ([]testing.TestResult, error)
}

// TaskRunner is an abstraction over various task-runners / execution environments.
// They are expected to implement the `taskRunner.Command` interface in turn, which is mapped to the Command type from
// `os/exec`
type TaskRunner interface {
	NewCommand(ctx context.Context, name string, args ...string) (exec.Command, error)
	GetExitStatusFromError(error) (int, error)
}
