// Package cli holds the main business logic in our CLI. This is mainly:
// 1. Triggering the right API calls based on the provided input parameters.
// 2. User-friendly logging
// However, this package _does not_ implement the actual terminal UI. That part is handled by `cmd/captain`.
package cli

import (
	"bufio"
	"context"
	"io"
	"mime"
	"path"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
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

func (s Service) parse(filepaths []string) (map[string]struct{}, error) {
	failedTests := make(map[string]struct{})

	for _, artifactPath := range filepaths {
		var matchingParser Parser

		fd, err := s.FileSystem.Open(artifactPath)
		if err != nil {
			return nil, s.logError(errors.NewSystemError("unable to open file: %s", err))
		}
		defer fd.Close()

		for _, parser := range s.Parsers {
			if err := parser.Parse(fd); err != nil {
				s.Log.Debug("unable to parse %q using %T: %s", artifactPath, parser, err)

				if _, err := fd.Seek(0, io.SeekStart); err != nil {
					return nil, s.logError(errors.NewSystemError("unable to read from file: %s", err))
				}

				continue
			}

			matchingParser = parser
			break
		}

		if matchingParser == nil {
			return nil, s.logError(errors.NewInputError("unable to parse %q with any of the available parser", artifactPath))
		}

		for matchingParser.NextTestCase() {
			if matchingParser.IsTestCaseFailed() {
				failedTests[matchingParser.TestCaseID()] = struct{}{}
			}
		}
	}

	return failedTests, nil
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

	if err := cmd.Start(); err != nil {
		return s.logError(errors.NewSystemError("unable to execute sub-command: %s", err))
	}

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
func (s Service) RunSuite(ctx context.Context, args []string, name, artifactGlob string) error {
	var wg sync.WaitGroup
	var quarantinedTestIDs []string

	if len(args) == 0 {
		return nil
	}

	// Fetch list of quarantined test IDs in the background
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		testIDs, err := s.API.GetQuarantinedTestIDs(egCtx)
		if err != nil {
			return errors.Wrap(err)
		}

		quarantinedTestIDs = testIDs
		return nil
	})

	// Run sub-command
	runErr := s.runCommand(ctx, args)
	if _, ok := errors.AsExecutionError(runErr); runErr != nil && !ok {
		return errors.Wrap(runErr)
	}

	// Return early if not artifacts were defined over the CLI. If there was an error during execution, the exit Code is
	// being passed along.
	if artifactGlob == "" {
		return runErr
	}

	artifacts, err := filepath.Glob(artifactGlob)
	if err != nil {
		return s.logError(errors.NewSystemError("unable to expand filepath glob: %s", err))
	}

	// Start uploading artifacts in the background
	wg.Add(1)
	go func() {
		// We ignore the error here since `UploadTestResults` will already log any errors. Furthermore, any errors here will
		// not affect the exit code.
		_ = s.UploadTestResults(ctx, name, artifacts)
		wg.Done()
	}()

	// Wait until list of quarantined tests was fetched. Ignore any errors.
	if err := eg.Wait(); err != nil {
		s.Log.Warnf("Unable to fetch list of quarantined tests from Captain: %s", err)
	}

	// Parse test artifacts
	failedTests, err := s.parse(artifacts)
	if err != nil {
		return s.logError(errors.Wrap(err))
	}

	// Remove quarantined IDs from list of failed tests
	for _, quarantinedTestID := range quarantinedTestIDs {
		delete(failedTests, quarantinedTestID)
	}

	// Wait until raw version of test results was uploaded
	wg.Wait()

	// Return original exit code in case there are failed tests.
	if runErr != nil && len(failedTests) > 0 {
		return runErr
	}

	return nil
}

// UploadTestResults is the implementation of `captain upload results`. It takes a number of filepaths, checks that
// files exist and uploads them using the provided API.
func (s Service) UploadTestResults(ctx context.Context, suiteName string, filepaths []string) error {
	newArtifacts := make([]api.Artifact, len(filepaths))

	for i, filePath := range filepaths {
		fd, err := s.FileSystem.Open(filePath)
		if err != nil {
			return s.logError(errors.NewSystemError("unable to open file: %s", err))
		}
		defer fd.Close()

		id, err := uuid.NewRandom()
		if err != nil {
			return s.logError(errors.NewInternalError("unable to generate new UUID: %s", err))
		}

		newArtifacts[i] = api.Artifact{
			ExternalID:   id,
			FD:           fd,
			Kind:         api.ArtifactKindTestResult,
			MimeType:     mime.TypeByExtension(path.Ext(filePath)),
			Name:         suiteName,
			OriginalPath: filePath,
			Parser:       api.ParserTypeRSpecJSON,
		}
	}

	if err := s.API.UploadArtifacts(ctx, newArtifacts); err != nil {
		return s.logError(errors.WithMessage("unable to upload test result: %s", err))
	}

	return nil
}
