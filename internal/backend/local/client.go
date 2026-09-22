package local

import (
	"context"
	"encoding/base64"
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

func (c Client) GetTestTimingManifest(
	_ context.Context, _ string, granularities ...string,
) ([]testing.TestFileTiming, error) {
	timings := c.Timings
	if len(granularities) > 0 && granularities[0] != "" && granularities[0] != "file" {
		var err error
		timings, err = c.readEntityTimings(granularities[0])
		if err != nil {
			return nil, err
		}
	}
	testTimings := make([]testing.TestFileTiming, 0, len(timings))

	for file, duration := range timings {
		testTimings = append(testTimings, testing.TestFileTiming{
			Filepath: file,
			Duration: duration,
		})
	}

	return testTimings, nil
}

func (c Client) entityTimingsPath(granularity string) string {
	return c.timingsPath + "." + base64.RawURLEncoding.EncodeToString([]byte(granularity))
}

func (c Client) readEntityTimings(granularity string) (map[string]time.Duration, error) {
	timings := make(map[string]time.Duration)
	file, err := c.fs.Open(c.entityTimingsPath(granularity))
	if errors.Is(err, os.ErrNotExist) {
		return timings, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer file.Close()
	if err := yaml.NewDecoder(file).Decode(&timings); err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.WithStack(err)
	}
	return timings, nil
}

func (c Client) UploadTimingManifest(_ context.Context, _ string, manifest v1.TimingManifest) error {
	timings := c.Timings
	path := c.timingsPath
	if manifest.Granularity != "file" {
		var err error
		timings, err = c.readEntityTimings(manifest.Granularity)
		if err != nil {
			return err
		}
		path = c.entityTimingsPath(manifest.Granularity)
	}
	if timings == nil {
		timings = make(map[string]time.Duration)
	}
	for _, timing := range manifest.Timings {
		timings[timing.Identifier] = timing.Duration
	}
	file, err := c.fs.Create(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()
	return errors.WithStack(yaml.NewEncoder(file).Encode(timings))
}

func (c Client) GetRunConfiguration(_ context.Context, _ string) (backend.RunConfiguration, error) {
	return makeRunConfiguration(c.Flakes, c.Quarantines, c.quarantinesTime)
}

func (c Client) GetQuarantinedTests(_ context.Context, _ string) ([]backend.Test, error) {
	quarantinedTests := make([]backend.Test, len(c.Quarantines))

	for i, quarantine := range c.Quarantines {
		identity, strict := NewMapFromYAML(quarantine).withoutKey("strict")

		quarantinedTests[i] = backend.Test{
			CompositeIdentifier: newCompositeID(identity),
			IdentityComponents:  identity.Order,
			StrictIdentity:      strict == "true",
		}
	}

	return quarantinedTests, nil
}

func (c Client) UpdateTestResults(
	ctx context.Context,
	suite string,
	testResults v1.TestResults,
) ([]backend.TestResultsUploadResult, error) {
	if len(testResults.TimingManifests) > 0 {
		for _, manifest := range testResults.TimingManifests {
			if err := c.UploadTimingManifest(ctx, suite, manifest); err != nil {
				return nil, err
			}
		}
		testResults.Tests = nil
	}
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
