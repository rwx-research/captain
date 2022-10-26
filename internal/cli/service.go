// Package cli holds the main business logic in our CLI. This is mainly:
// 1. Triggering the right API calls based on the provided input parameters.
// 2. User-friendly logging
// However, this package _does not_ implement the actual terminal UI. That part is handled by `cmd/captain`.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/parsers"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// Service is the main CLI service.
type Service struct {
	API        APIClient
	Log        *zap.SugaredLogger
	FileSystem FileSystem
	TaskRunner TaskRunner
	Parsers    []Parser
}

func (s Service) logError(err error) error {
	s.Log.Errorf(err.Error())
	return err
}

// Parse parses the files supplied in `filepaths` and prints them as formatted JSON to stdout.
func (s Service) Parse(ctx context.Context, filepaths []string) error {
	results, err := s.parse(filepaths)
	if err != nil {
		// Error was already logged in `parse`
		return errors.Wrap(err)
	}

	rawOutput := make(map[string]testResult)
	for id, result := range results {
		rawOutput[id] = testResult{
			Description:   result.Description,
			Duration:      result.Duration,
			Status:        testStatus(result.Status),
			StatusMessage: result.StatusMessage,
		}
	}

	formattedOutput, err := json.MarshalIndent(rawOutput, "", "  ")
	if err != nil {
		return s.logError(errors.NewSystemError("unable to output test results as JSON: %s", err))
	}

	s.Log.Infoln(string(formattedOutput))

	return nil
}

func (s Service) parse(filepaths []string) (map[string]testing.TestResult, error) {
	results := make(map[string]testing.TestResult)

	for _, artifactPath := range filepaths {
		var result map[string]testing.TestResult

		s.Log.Debugf("Attempting to parse %q", artifactPath)

		fd, err := s.FileSystem.Open(artifactPath)
		if err != nil {
			return nil, s.logError(errors.NewSystemError("unable to open file: %s", err))
		}
		defer fd.Close()

		for _, parser := range s.Parsers {
			result, err = parser.Parse(fd)
			if err != nil {
				s.Log.Debugf("unable to parse %q using %T: %s", artifactPath, parser, err)

				if _, err := fd.Seek(0, io.SeekStart); err != nil {
					return nil, s.logError(errors.NewSystemError("unable to read from file: %s", err))
				}

				continue
			}

			s.Log.Debugf("Successfully parsed %q using %T", artifactPath, parser)
			break
		}

		if result == nil {
			return nil, s.logError(errors.NewInputError("unable to parse %q with any of the available parser", artifactPath))
		}

		for id, testResult := range result {
			results[id] = testResult
		}
	}

	return results, nil
}

// PrintVersion prints the CLI version
func (s Service) PrintVersion() {
	s.Log.Infoln(captain.Version)
}

func (s Service) runCommand(ctx context.Context, args []string) error {
	var cmd exec.Command
	var err error

	if len(args) == 1 {
		cmd, err = s.TaskRunner.NewCommand(ctx, args[0])
	} else {
		cmd, err = s.TaskRunner.NewCommand(ctx, args[0], args[1:]...)
	}
	if err != nil {
		return s.logError(errors.NewSystemError("unable to spawn sub-process: %s", err))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.Log.Warnf("unable to attach to stdout of sub-process: %s", err)
	}

	s.Log.Debugf("Executing %q", strings.Join(args, " "))
	if err := cmd.Start(); err != nil {
		return s.logError(errors.NewSystemError("unable to execute sub-command: %s", err))
	}
	defer s.Log.Debugf("Finished executing %q", strings.Join(args, " "))

	if stdout != nil {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			s.Log.Info(scanner.Text())
		}
	}

	if err := cmd.Wait(); err != nil {
		if code, e := s.TaskRunner.GetExitStatusFromError(err); e == nil {
			return errors.NewExecutionError(code, "encountered error during execution of sub-process")
		}

		return s.logError(errors.NewSystemError("Error during program execution: %s", err))
	}

	return nil
}

// RunSuite runs the specified build- or test-suite and optionally uploads the resulting artifact.
func (s Service) RunSuite(ctx context.Context, cfg RunConfig) error {
	var wg sync.WaitGroup
	var quarantinedTestIDs []string

	if len(cfg.Args) == 0 {
		s.Log.Debug("No arguments provided to `RunSuite`")
		return nil
	}

	// Fetch list of quarantined test IDs in the background
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		testIDs, err := s.API.GetQuarantinedTestIDs(egCtx)
		if err != nil {
			return errors.Wrap(err)
		}

		if len(testIDs) == 0 {
			s.Log.Debug("No quarantined tests defined in Captain")
		}

		quarantinedTestIDs = testIDs
		return nil
	})

	// Run sub-command
	runErr := s.runCommand(ctx, cfg.Args)
	if _, ok := errors.AsExecutionError(runErr); runErr != nil && !ok {
		return errors.Wrap(runErr)
	}

	// Return early if not artifacts were defined over the CLI. If there was an error during execution, the exit Code is
	// being passed along.
	if cfg.ArtifactGlob == "" {
		s.Log.Debug("No artifact path provided, quitting")
		return runErr
	}

	artifacts, err := filepath.Glob(cfg.ArtifactGlob)
	if err != nil {
		return s.logError(errors.NewSystemError("unable to expand filepath glob: %s", err))
	}

	// Start uploading artifacts in the background
	wg.Add(1)
	var uploadError error

	go func() {
		// We ignore the error here since `UploadTestResults` will already log any errors. Furthermore, any errors here will
		// not affect the exit code.
		uploadError = s.UploadTestResults(ctx, cfg.SuiteName, artifacts)
		wg.Done()
	}()

	// Wait until list of quarantined tests was fetched. Ignore any errors.
	if err := eg.Wait(); err != nil {
		s.Log.Warnf("Unable to fetch list of quarantined tests from Captain: %s", err)
	}

	// Parse test artifacts
	testResults, err := s.parse(artifacts)
	if err != nil {
		return s.logError(errors.Wrap(err))
	}

	failedTests := make(map[string]struct{})
	for id, testResult := range testResults {
		if testResult.Status == testing.TestStatusFailed {
			failedTests[id] = struct{}{}
		}
	}

	// This protects against an edge-case where the test-suite fails before writing any results to an artifact.
	if runErr != nil && len(failedTests) == 0 {
		return runErr
	}

	// Remove quarantined IDs from list of failed tests
	for _, quarantinedTestID := range quarantinedTestIDs {
		s.Log.Debugf("ignoring test results for %q due to quarantine", quarantinedTestID)
		delete(failedTests, quarantinedTestID)
	}

	// Wait until raw version of test results was uploaded
	wg.Wait()

	// Return original exit code in case there are failed tests.
	if runErr != nil && len(failedTests) > 0 {
		return runErr
	}

	if uploadError != nil && cfg.FailOnUploadError {
		return uploadError
	}

	return nil
}

// UploadTestResults is the implementation of `captain upload results`. It takes a number of filepaths, checks that
// files exist and uploads them using the provided API.
func (s Service) UploadTestResults(ctx context.Context, suiteName string, filepaths []string) error {
	newArtifacts := make([]api.Artifact, len(filepaths))

	if len(filepaths) == 0 {
		s.Log.Debug("No paths to test results provided")
		return nil
	}

	for i, filePath := range filepaths {
		s.Log.Debugf("Attempting to upload %q to Captain", filePath)

		fd, err := s.FileSystem.Open(filePath)
		if err != nil {
			return s.logError(errors.NewSystemError("unable to open file: %s", err))
		}
		defer fd.Close()

		id, err := uuid.NewRandom()
		if err != nil {
			return s.logError(errors.NewInternalError("unable to generate new UUID: %s", err))
		}

		s.Log.Debugf("Attempting to parse %q", filePath)
		var parserType api.ParserType

		// Re-parsing the files here is not great. This will be removed once the API doesn't expect
		// a parser type anymore.
		for _, parser := range s.Parsers {
			_, parseErr := parser.Parse(fd)

			// We always have to seek back to the beginning since we'll re-read the file upon upload.
			if _, err := fd.Seek(0, io.SeekStart); err != nil {
				return s.logError(errors.NewSystemError("unable to read from file: %s", err))
			}

			if parseErr != nil {
				s.Log.Debugf("unable to parse %q using %T: %s", filePath, parser, parseErr)
				continue
			}

			s.Log.Debugf("Successfully parsed %q using %T", filePath, parser)

			switch parser.(type) {
			case *parsers.Jest:
				parserType = api.ParserTypeJestJSON
			case *parsers.JUnit:
				parserType = api.ParserTypeJUnitXML
			case *parsers.RSpecV3:
				parserType = api.ParserTypeRSpecJSON
			case *parsers.XUnitDotNetV2:
				parserType = api.ParserTypeXUnitXML
			}

			break
		}

		newArtifacts[i] = api.Artifact{
			ExternalID:   id,
			FD:           fd,
			Kind:         api.ArtifactKindTestResult,
			MimeType:     mime.TypeByExtension(path.Ext(filePath)),
			Name:         suiteName,
			OriginalPath: filePath,
			Parser:       parserType,
		}
	}

	if err := s.API.UploadArtifacts(ctx, newArtifacts); err != nil {
		return s.logError(errors.WithMessage("unable to upload test result: %s", err))
	}

	return nil
}
