// Package cli holds the main business logic in our CLI. This is mainly:
// 1. Triggering the right API calls based on the provided input parameters.
// 2. User-friendly logging
// However, this package _does not_ implement the actual terminal UI. That part is handled by `cmd/captain`.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

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

	rawOutput := make([]testResult, 0)
	for _, result := range results {
		rawOutput = append(rawOutput, testResult{
			Description:   result.Description,
			Duration:      result.Duration,
			Status:        testStatus(result.Status),
			StatusMessage: result.StatusMessage,
			Meta:          result.Meta,
		})
	}

	formattedOutput, err := json.MarshalIndent(rawOutput, "", "  ")
	if err != nil {
		return s.logError(errors.NewSystemError("unable to output test results as JSON: %s", err))
	}

	s.Log.Infoln(string(formattedOutput))

	return nil
}

func (s Service) parse(filepaths []string) ([]testing.TestResult, error) {
	results := make([]testing.TestResult, 0)

	for _, testResultsFilePath := range filepaths {
		var result []testing.TestResult

		s.Log.Debugf("Attempting to parse %q", testResultsFilePath)

		fd, err := s.FileSystem.Open(testResultsFilePath)
		if err != nil {
			return nil, s.logError(errors.NewSystemError("unable to open file: %s", err))
		}
		defer fd.Close()

		for _, parser := range s.Parsers {
			result, err = parser.Parse(fd)
			if err != nil {
				s.Log.Debugf("unable to parse %q using %T: %s", testResultsFilePath, parser, err)

				if _, err := fd.Seek(0, io.SeekStart); err != nil {
					return nil, s.logError(errors.NewSystemError("unable to read from file: %s", err))
				}

				continue
			}

			s.Log.Debugf("Successfully parsed %q using %T", testResultsFilePath, parser)
			break
		}

		if result == nil {
			return nil, s.logError(
				errors.NewInputError("unable to parse %q with any of the available parser", testResultsFilePath),
			)
		}

		results = append(results, result...)
	}

	return results, nil
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

func (s Service) isQuarantined(failedTest testing.TestResult, quarantinedTestCases []api.QuarantinedTestCase) bool {
	for _, quarantinedTestCase := range quarantinedTestCases {
		compositeIdentifier, err := failedTest.Identify(
			quarantinedTestCase.IdentityComponents,
			quarantinedTestCase.StrictIdentity,
		)
		if err != nil {
			s.Log.Debugf("%v does not quarantine %v because %v", quarantinedTestCase, failedTest, err.Error())
			continue
		}

		if compositeIdentifier == quarantinedTestCase.CompositeIdentifier {
			s.Log.Debugf(
				"%v quarantines %v because they share the same composite identifier (%v)",
				quarantinedTestCase,
				failedTest,
				compositeIdentifier,
			)
			return true
		}
	}

	return false
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}

// RunSuite runs the specified build- or test-suite and optionally uploads the resulting test results file.
func (s Service) RunSuite(ctx context.Context, cfg RunConfig) error {
	var wg sync.WaitGroup
	var quarantinedTestCases []api.QuarantinedTestCase

	if len(cfg.Args) == 0 {
		s.Log.Debug("No arguments provided to `RunSuite`")
		return nil
	}

	// Fetch list of quarantined test IDs in the background
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		testCases, err := s.API.GetQuarantinedTestCases(egCtx, cfg.SuiteID)
		if err != nil {
			return errors.Wrap(err)
		}

		if len(testCases) == 0 {
			s.Log.Debug("No quarantined tests defined in Captain")
		}

		quarantinedTestCases = testCases
		return nil
	})

	// Run sub-command
	runErr := s.runCommand(ctx, cfg.Args)
	if _, ok := errors.AsExecutionError(runErr); runErr != nil && !ok {
		return errors.Wrap(runErr)
	}

	// Return early if not testResultsFiles were defined over the CLI. If there was an error
	// during execution, the exit Code is being passed along.
	if cfg.TestResultsFileGlob == "" {
		s.Log.Debug("No testResultsFile path provided, quitting")
		return runErr
	}

	testResultsFiles, err := s.FileSystem.Glob(cfg.TestResultsFileGlob)
	if err != nil {
		return s.logError(errors.NewSystemError("unable to expand filepath glob: %s", err))
	}

	// Start uploading testResultsFiles in the background
	wg.Add(1)
	var uploadResults []api.TestResultsUploadResult
	var uploadError error

	go func() {
		// We ignore the error here since `UploadTestResults` will already log any errors. Furthermore, any errors here will
		// not affect the exit code.
		uploadResults, uploadError = s.UploadTestResults(ctx, cfg.SuiteID, testResultsFiles)
		wg.Done()
	}()

	// Wait until list of quarantined tests was fetched. Ignore any errors.
	if err := eg.Wait(); err != nil {
		s.Log.Warnf("Unable to fetch list of quarantined tests from Captain: %s", err)
	}

	// Parse test testResultsFiles
	testResults, err := s.parse(testResultsFiles)
	if err != nil {
		return s.logError(errors.Wrap(err))
	}

	failedTests := make([]testing.TestResult, 0)
	for _, testResult := range testResults {
		if testResult.Status == testing.TestStatusFailed {
			failedTests = append(failedTests, testResult)
		}
	}

	// This protects against an edge-case where the test-suite fails before writing any results to a test results file.
	if runErr != nil && len(failedTests) == 0 {
		return runErr
	}

	quarantinedFailedTests := make([]testing.TestResult, 0)
	unquarantinedFailedTests := make([]testing.TestResult, 0)
	for _, failedTest := range failedTests {
		s.Log.Debugf("attempting to quarantine failed test: %v", failedTest)

		if s.isQuarantined(failedTest, quarantinedTestCases) {
			s.Log.Debugf("quarantined failed test: %v", failedTest)
			quarantinedFailedTests = append(quarantinedFailedTests, failedTest)
		} else {
			s.Log.Debugf("did not quarantine failed test: %v", failedTest)
			unquarantinedFailedTests = append(unquarantinedFailedTests, failedTest)
		}
	}

	// Wait until raw version of test results was uploaded
	wg.Wait()

	// Display detailed output if necessary
	hasUploadResults := len(uploadResults) > 0
	hasQuarantinedFailedTests := len(quarantinedFailedTests) > 0
	hasDetails := hasUploadResults || hasQuarantinedFailedTests

	if hasDetails {
		s.Log.Infoln(strings.Repeat("-", 80))
		s.Log.Infoln(fmt.Sprintf("%v Captain %v", strings.Repeat("-", 40-4-1), strings.Repeat("-", 40-3-1)))
		s.Log.Infoln(strings.Repeat("-", 80))
	}

	if hasUploadResults {
		s.Log.Infoln(
			fmt.Sprintf(
				"\nFound %v test result %v:",
				len(uploadResults),
				pluralize(len(uploadResults), "file", "files"),
			),
		)

		for _, uploadResult := range uploadResults {
			if uploadResult.Uploaded {
				s.Log.Infoln(fmt.Sprintf("- Uploaded %v", uploadResult.OriginalPath))
			} else {
				s.Log.Infoln(fmt.Sprintf("- Unable to upload %v", uploadResult.OriginalPath))
			}
		}
	}

	if hasQuarantinedFailedTests {
		s.Log.Infoln(
			fmt.Sprintf(
				"\n%v of %v %v under quarantine:",
				len(quarantinedFailedTests),
				len(failedTests),
				pluralize(len(failedTests), "failure", "failures"),
			),
		)

		for _, quarantinedFailedTest := range quarantinedFailedTests {
			s.Log.Infoln(fmt.Sprintf("- %v", quarantinedFailedTest.Description))
		}
	}

	if hasDetails && len(unquarantinedFailedTests) > 0 {
		s.Log.Infoln(
			fmt.Sprintf(
				"\n%v remaining actionable %v",
				len(unquarantinedFailedTests),
				pluralize(len(unquarantinedFailedTests), "failure", "failures"),
			),
		)

		for _, unquarantinedFailedTests := range unquarantinedFailedTests {
			s.Log.Infoln(fmt.Sprintf("- %v", unquarantinedFailedTests.Description))
		}
	}

	// Return original exit code in case there are failed tests.
	if runErr != nil && len(unquarantinedFailedTests) > 0 {
		return runErr
	}

	if uploadError != nil && cfg.FailOnUploadError {
		return uploadError
	}

	return nil
}

// Partition splits a glob of test filepaths using decreasing first fit backed by a timing manifest from captain.
func (s Service) Partition(ctx context.Context, cfg PartitionConfig) error {
	err := cfg.Validate()
	if err != nil {
		return errors.Wrap(err)
	}
	fileTimings, err := s.API.GetTestTimingManifest(ctx, cfg.SuiteID)
	if err != nil {
		return errors.Wrap(err)
	}

	testFilePaths, err := s.FileSystem.GlobMany(cfg.TestFilePaths)
	if err != nil {
		return s.logError(errors.NewSystemError("unable to expand filepath glob: %s", err))
	}

	// Compare expanded client file paths w/ expanded server file paths
	// taking care to always use the client path and sort by duration desc
	fileTimingMatches := make([]testing.FileTimingMatch, 0)
	unmatchedFilepaths := make([]string, 0)
	for _, clientTestFile := range testFilePaths {
		match := false
		var fileTimingMatch testing.FileTimingMatch
		clientExpandedFilepath, err := filepath.Abs(clientTestFile)
		if err != nil {
			s.Log.Warnf("failed to expand path of test file: %s", clientTestFile)
			unmatchedFilepaths = append(unmatchedFilepaths, clientTestFile)
			continue
		}

		for _, serverTiming := range fileTimings {
			serverExpandedFilepath, err := filepath.Abs(serverTiming.Filepath)
			if err != nil {
				s.Log.Warnf("failed to expand filepath of timing file: %s", serverTiming.Filepath)
				break
			}
			if clientExpandedFilepath == serverExpandedFilepath {
				match = true
				fileTimingMatch = testing.FileTimingMatch{
					FileTiming:     serverTiming,
					ClientFilepath: clientTestFile,
				}
				break
			}
		}
		if match {
			fileTimingMatches = append(fileTimingMatches, fileTimingMatch)
		} else {
			unmatchedFilepaths = append(unmatchedFilepaths, clientTestFile)
		}
	}
	sort.Slice(fileTimingMatches, func(i, j int) bool {
		return fileTimingMatches[i].Duration() > fileTimingMatches[j].Duration()
	})

	if len(fileTimingMatches) == 0 {
		s.Log.Errorln("No test file timings were matched. Using naive round-robin strategy.")
	}

	partitions := make([]testing.TestPartition, 0)
	var totalCapacity time.Duration
	for _, fileTimingMatch := range fileTimingMatches {
		totalCapacity += fileTimingMatch.Duration()
	}
	partitionCapacity := totalCapacity / time.Duration(cfg.TotalPartitions)

	s.Log.Debugf("Total Capacity: %s", totalCapacity)
	s.Log.Debugf("Target Partition Capacity: %s", partitionCapacity)

	for i := 0; i < cfg.TotalPartitions; i++ {
		partitions = append(partitions, testing.TestPartition{
			Index:             i,
			TestFilePaths:     make([]string, 0),
			RemainingCapacity: partitionCapacity,
			TotalCapacity:     partitionCapacity,
		})
	}

	for _, fileTimingMatch := range fileTimingMatches {
		fits, partition := partitionWithFirstFit(partitions, fileTimingMatch)
		if fits {
			partition = partition.Add(fileTimingMatch)
			partitions[partition.Index] = partition
			s.Log.Debugf("%s: Assigned %s using first fit strategy", partition, fileTimingMatch)
			continue
		}
		partition = partitionWithMostRemainingCapacity(partitions)
		partition = partition.Add(fileTimingMatch)
		partitions[partition.Index] = partition
		s.Log.Debugf("%s: Assigned %s using most remaining capacity strategy", partition, fileTimingMatch)
	}

	for i, testFilepath := range unmatchedFilepaths {
		partition := partitions[i%len(partitions)]
		partitions[partition.Index] = partition.AddFilePath(testFilepath)
		s.Log.Debugf("%s: Assigned '%s' using round robin strategy", partition, testFilepath)
	}

	activePartition := partitions[cfg.PartitionIndex]
	s.Log.Infoln(strings.Join(activePartition.TestFilePaths, " "))

	return nil
}

func partitionWithFirstFit(
	partitions []testing.TestPartition,
	fileTimingMatch testing.FileTimingMatch,
) (fit bool, result testing.TestPartition) {
	for _, p := range partitions {
		if p.RemainingCapacity >= fileTimingMatch.Duration() {
			return true, p
		}
	}
	return false, result
}

func partitionWithMostRemainingCapacity(partitions []testing.TestPartition) testing.TestPartition {
	result := partitions[0]
	for i := 1; i < len(partitions); i++ {
		p := partitions[i]
		if p.RemainingCapacity > result.RemainingCapacity {
			result = p
		}
	}
	return result
}

// UploadTestResults is the implementation of `captain upload results`. It takes a number of filepaths, checks that
// files exist and uploads them using the provided API.
func (s Service) UploadTestResults(
	ctx context.Context,
	testSuiteID string,
	filepaths []string,
) ([]api.TestResultsUploadResult, error) {
	if len(filepaths) == 0 {
		s.Log.Debug("No paths to test results provided")
		return nil, nil
	}

	expandedFilepaths, err := s.FileSystem.GlobMany(filepaths)
	newTestResultsFiles := make([]api.TestResultsFile, len(expandedFilepaths))

	if err != nil {
		return nil, s.logError(errors.NewSystemError("unable to expand filepath glob: %s", err))
	}

	for i, filePath := range expandedFilepaths {
		s.Log.Debugf("Attempting to upload %q to Captain", filePath)

		fd, err := s.FileSystem.Open(filePath)
		if err != nil {
			return nil, s.logError(errors.NewSystemError("unable to open file: %s", err))
		}
		defer fd.Close()

		id, err := uuid.NewRandom()
		if err != nil {
			return nil, s.logError(errors.NewInternalError("unable to generate new UUID: %s", err))
		}

		s.Log.Debugf("Attempting to parse %q", filePath)
		var parserType api.ParserType

		// Re-parsing the files here is not great. This will be removed once the API doesn't expect
		// a parser type anymore.
		for _, parser := range s.Parsers {
			_, parseErr := parser.Parse(fd)

			// We always have to seek back to the beginning since we'll re-read the file upon upload.
			if _, err := fd.Seek(0, io.SeekStart); err != nil {
				return nil, s.logError(errors.NewSystemError("unable to read from file: %s", err))
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

		newTestResultsFiles[i] = api.TestResultsFile{
			ExternalID:   id,
			FD:           fd,
			OriginalPath: filePath,
			Parser:       parserType,
		}
	}

	uploadResults, err := s.API.UploadTestResults(ctx, testSuiteID, newTestResultsFiles)
	if err != nil {
		return nil, s.logError(errors.WithMessage("unable to upload test result: %s", err))
	}

	return uploadResults, nil
}
