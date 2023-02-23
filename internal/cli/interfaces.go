package cli

import (
	"context"
	"os"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/testing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
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
	Create(filePath string) (fs.File, error)
	Open(name string) (fs.File, error)
	Glob(pattern string) ([]string, error)
	GlobMany(patterns []string) ([]string, error)
	Remove(name string) error
	Rename(oldname string, newname string) error
	Stat(name string) (os.FileInfo, error)
	TempDir() string
}

// Reporter is a function that writes test results to a file. Different reporters implement different encodings.
type Reporter func(fs.File, v1.TestResults) error

// TaskRunner is an abstraction over various task-runners / execution environments.
// They are expected to implement the `taskRunner.Command` interface in turn, which is mapped to the Command type from
// `os/exec`
type TaskRunner interface {
	NewCommand(ctx context.Context, name string, args []string, environ []string) (exec.Command, error)
	GetExitStatusFromError(error) (int, error)
}
