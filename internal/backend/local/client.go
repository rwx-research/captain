package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/testing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type Client struct {
	fs              fs.FileSystem
	Flakes          []yaml.Node
	flakesPath      string
	Quarantines     []yaml.Node
	quarantinesPath string
	quarantinesTime time.Time
	Timings         map[string]time.Duration
	timingsPath     string
}

func NewClient(fileSystem fs.FileSystem, flakesPath, quarantinesPath, timingsPath string) (Client, error) {
	c := Client{
		fs:              fileSystem,
		flakesPath:      flakesPath,
		quarantinesPath: quarantinesPath,
		Timings:         make(map[string]time.Duration),
		timingsPath:     timingsPath,
	}

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

		if err = yaml.NewEncoder(fd).Encode(v); err != nil {
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

		if path == quarantinesPath {
			info, err := fd.Stat()
			if err != nil {
				return errors.WithStack(err)
			}

			c.quarantinesTime = info.ModTime()
		}

		if err := yaml.NewDecoder(fd).Decode(v); err != nil && !errors.Is(err, io.EOF) {
			return errors.WithStack(err)
		}

		return nil
	}

	if err := read(flakesPath, &c.Flakes); err != nil {
		return c, errors.WithStack(err)
	}

	if err := read(quarantinesPath, &c.Quarantines); err != nil {
		return c, errors.WithStack(err)
	}

	if err := read(timingsPath, &c.Timings); err != nil {
		return c, errors.WithStack(err)
	}

	return c, nil
}

func (c Client) Flush() error {
	write := func(filepath string, data any) error {
		file, err := c.fs.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return errors.NewSystemError("unable to open %q: %s", filepath, err)
		}
		defer file.Close()

		encoder := yaml.NewEncoder(file)
		if err := encoder.Encode(data); err != nil {
			return errors.NewSystemError("unable to write to %q: %s", filepath, err)
		}

		return nil
	}

	if err := write(c.flakesPath, c.Flakes); err != nil {
		return err
	}

	return write(c.quarantinesPath, c.Quarantines)
}

func (c Client) GetTestTimingManifest(_ context.Context, _ string) ([]testing.TestFileTiming, error) {
	testTimings := make([]testing.TestFileTiming, 0)

	for file, duration := range c.Timings {
		testTimings = append(testTimings, testing.TestFileTiming{
			Filepath: file,
			Duration: duration,
		})
	}

	return testTimings, nil
}

func (c Client) GetRunConfiguration(_ context.Context, _ string) (backend.RunConfiguration, error) {
	return makeRunConfiguration(c.Flakes, c.Quarantines, c.quarantinesTime)
}

func (c Client) UpdateTestResults(
	_ context.Context,
	_ string,
	testResults v1.TestResults,
	_ bool,
) ([]backend.TestResultsUploadResult, error) {
	if c.Timings == nil {
		c.Timings = make(map[string]time.Duration)
	}

	newTimings := make(map[string]time.Duration)

	for _, test := range testResults.Tests {
		if test.Location != nil && test.Attempt.Duration != nil {
			testDuration, ok := newTimings[test.Location.File]
			if ok {
				testDuration += *test.Attempt.Duration
			} else {
				testDuration = *test.Attempt.Duration
			}
			newTimings[test.Location.File] = testDuration
		}
	}

	for file, duration := range newTimings {
		c.Timings[file] = duration
	}

	timingsFile, err := c.fs.OpenFile(c.timingsPath, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return nil, errors.NewSystemError("unable to open %q: %s", c.timingsPath, err)
	}
	defer timingsFile.Close()

	timingsEncoder := yaml.NewEncoder(timingsFile)
	if err := timingsEncoder.Encode(c.Timings); err != nil {
		return nil, errors.NewSystemError("unable to write to %q: %s", c.timingsPath, err)
	}

	originalPaths := make([]string, len(testResults.DerivedFrom))
	for i, result := range testResults.DerivedFrom {
		originalPaths[i] = result.OriginalFilePath
	}

	return []backend.TestResultsUploadResult{{
		OriginalPaths: originalPaths,
		Uploaded:      true,
	}}, nil
}
