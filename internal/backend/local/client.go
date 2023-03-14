package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/testing"
)

type Client struct {
	RunConfiguration backend.RunConfiguration
	Timings          []testing.TestFileTiming `json:",omitempty"`
}

func NewClient(fileSystem fs.FileSystem, runConfigPath, timingsPath string) (Client, error) {
	var c Client

	openOrCreate := func(path string, v any) (fs.File, error) {
		fd, err := fileSystem.Open(path)
		if err == nil {
			return fd, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to open %q", path))
		}

		fd, err = fileSystem.Create(path)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to create %q", path))
		}

		if err = json.NewEncoder(fd).Encode(v); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to write to %q", path))
		}

		// We expect to read from this file again, so seek back to the start
		if _, err = fd.Seek(0, io.SeekStart); err != nil {
			return nil, errors.WithStack(err)
		}

		return fd, nil
	}

	read := func(path string, v any) error {
		fd, err := openOrCreate(path, v)
		if err != nil {
			return errors.WithStack(err)
		}
		defer fd.Close()

		if err := json.NewDecoder(fd).Decode(v); err != nil {
			return errors.WithStack(err)
		}

		return nil
	}

	if err := read(runConfigPath, &c.RunConfiguration); err != nil {
		fmt.Println("one", err)
		return c, errors.WithStack(err)
	}

	if err := read(timingsPath, &c.Timings); err != nil {
		fmt.Println("two", err)
		return c, errors.WithStack(err)
	}

	return c, nil
}

func (c Client) GetRunConfiguration(ctx context.Context, suiteID string) (backend.RunConfiguration, error) {
	return c.RunConfiguration, nil
}

func (c Client) GetTestTimingManifest(ctx context.Context, suiteID string) ([]testing.TestFileTiming, error) {
	return c.Timings, nil
}

func (c Client) UploadTestResults(
	ctx context.Context,
	suiteID string,
	testResultsFiles []backend.TestResultsFile,
) ([]backend.TestResultsUploadResult, error) {
	return nil, errors.NewConfigurationError("Uploading test restuls requires a Captain Cloud subscription.")
}
