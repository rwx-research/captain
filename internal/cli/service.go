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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/testing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// Service is the main CLI service.
type Service struct {
	API        APIClient
	Log        *zap.SugaredLogger
	FileSystem FileSystem
	TaskRunner TaskRunner
	Parsers    []parsing.Parser
}

func (s Service) logError(err error) error {
	s.Log.Errorf(err.Error())
	return err
}

// Parse parses the files supplied in `filepaths` and prints them as formatted JSON to stdout.
func (s Service) Parse(ctx context.Context, filepaths []string) error {
	results, err := s.parse(filepaths)
	if err != nil {
		return s.logError(errors.WithStack(err))
	}

	for _, result := range results {
		newOutput, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return s.logError(errors.NewInternalError("Unable to output test results as JSON: %s", err))
		}
		s.Log.Infoln(string(newOutput))
	}

	return nil
}

func (s Service) parse(filepaths []string) ([]v1.TestResults, error) {
	allResults := make([]v1.TestResults, 0)

	for _, testResultsFilePath := range filepaths {
		s.Log.Debugf("Attempting to parse %q", testResultsFilePath)

		fd, err := s.FileSystem.Open(testResultsFilePath)
		if err != nil {
			return nil, errors.NewSystemError("unable to open file: %s", err)
		}
		defer fd.Close()

		results, err := parsing.Parse(fd, s.Parsers, s.Log)
		if err != nil {
			return nil, errors.NewInputError("Unable to parse %q with the available parsers", testResultsFilePath)
		}

		allResults = append(allResults, *results)
	}

	return v1.Merge(allResults), nil
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

func (s Service) isQuarantined(failedTest v1.Test, quarantinedTestCases []api.QuarantinedTestCase) bool {
	for _, quarantinedTestCase := range quarantinedTestCases {
		compositeIdentifier, err := failedTest.Identify(
			quarantinedTestCase.IdentityComponents,
			quarantinedTestCase.StrictIdentity,
		)
		if err != nil {
			s.Log.Debugf("%v does not identify %v because %v", quarantinedTestCase, failedTest, err.Error())
			continue
		}

		if compositeIdentifier != quarantinedTestCase.CompositeIdentifier {
			s.Log.Debugf(
				"%v does not quarantine %v because they have different composite identifiers (%v != %v)",
				quarantinedTestCase,
				failedTest,
				quarantinedTestCase.CompositeIdentifier,
				compositeIdentifier,
			)
			continue
		}

		s.Log.Debugf(
			"%v quarantines %v because they share the same composite identifier (%v)",
			quarantinedTestCase,
			failedTest,
			compositeIdentifier,
		)
		return true
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
	if len(cfg.Args) == 0 {
		s.Log.Debug("No arguments provided to `RunSuite`")
		return nil
	}

	// Fetch list of quarantined test IDs in the background
	var quarantinedTestCases []api.QuarantinedTestCase
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		testCases, err := s.API.GetQuarantinedTestCases(egCtx, cfg.SuiteID)
		if err != nil {
			return errors.WithStack(err)
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
		return errors.WithStack(runErr)
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

	// Wait until list of quarantined tests was fetched. Ignore any errors.
	if err := eg.Wait(); err != nil {
		s.Log.Warnf("Unable to fetch list of quarantined tests from Captain: %s", err)
	}

	// Parse testResultsFiles
	testResults, err := s.parse(testResultsFiles)
	if err != nil {
		return s.logError(errors.WithStack(err))
	}

	quarantinedFailedTests := make([]v1.Test, 0)
	unquarantinedFailedTests := make([]v1.Test, 0)
	for i, testResult := range testResults {
		for i, test := range testResult.Tests {
			if !s.isQuarantined(test, quarantinedTestCases) {
				if test.Attempt.Status.Kind == v1.TestStatusFailed {
					s.Log.Debugf("did not quarantine failed test: %v", test)
					unquarantinedFailedTests = append(unquarantinedFailedTests, test)
				}
				continue
			}

			testResult.Tests[i] = test.Quarantine()

			if test.Attempt.Status.Kind == v1.TestStatusFailed {
				s.Log.Debugf("quarantined failed test: %v", test)
				quarantinedFailedTests = append(quarantinedFailedTests, test)
			}
		}
		testResult.Summary = v1.NewSummary(testResult.Tests, testResult.OtherErrors)
		testResults[i] = testResult
	}

	// This protects against an edge-case where the test-suite fails before writing any results to a test results file.
	if runErr != nil && len(quarantinedFailedTests)+len(unquarantinedFailedTests) == 0 {
		return runErr
	}

	var uploadResults []api.TestResultsUploadResult
	var uploadError error
	// We ignore the error here since `UploadTestResults` will already log any errors. Furthermore, any errors here will
	// not affect the exit code.
	if len(testResults) > 0 {
		uploadResults, uploadError = s.performTestResultsUpload(ctx, cfg.SuiteID, testResults)
	} else {
		s.Log.Debugf("No test results were parsed. Globbed files: %v", testResultsFiles)
	}

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
		uploadedPaths := make([]string, 0)
		erroredPaths := make([]string, 0)
		for _, uploadResult := range uploadResults {
			if uploadResult.Uploaded {
				uploadedPaths = append(uploadedPaths, uploadResult.OriginalPaths...)
			} else {
				erroredPaths = append(erroredPaths, uploadResult.OriginalPaths...)
			}
		}
		testResultsFilesFound := len(uploadedPaths) + len(erroredPaths)

		s.Log.Infoln(
			fmt.Sprintf(
				"\nFound %v test result %v:",
				testResultsFilesFound,
				pluralize(testResultsFilesFound, "file", "files"),
			),
		)

		for _, uploadedPath := range uploadedPaths {
			s.Log.Infoln(fmt.Sprintf("- Uploaded %v", uploadedPath))
		}
		for _, erroredPath := range erroredPaths {
			s.Log.Infoln(fmt.Sprintf("- Unable to upload %v", erroredPath))
		}
	}

	if hasQuarantinedFailedTests {
		s.Log.Infoln(
			fmt.Sprintf(
				"\n%v of %v %v under quarantine:",
				len(quarantinedFailedTests),
				len(quarantinedFailedTests)+len(unquarantinedFailedTests),
				pluralize(len(quarantinedFailedTests)+len(unquarantinedFailedTests), "failure", "failures"),
			),
		)

		for _, quarantinedFailedTest := range quarantinedFailedTests {
			s.Log.Infoln(fmt.Sprintf("- %v", quarantinedFailedTest.Name))
		}
	}

	if hasDetails && hasQuarantinedFailedTests && len(unquarantinedFailedTests) > 0 {
		s.Log.Infoln(
			fmt.Sprintf(
				"\n%v remaining actionable %v",
				len(unquarantinedFailedTests),
				pluralize(len(unquarantinedFailedTests), "failure", "failures"),
			),
		)

		for _, unquarantinedFailedTests := range unquarantinedFailedTests {
			s.Log.Infoln(fmt.Sprintf("- %v", unquarantinedFailedTests.Name))
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
		return errors.WithStack(err)
	}
	fileTimings, err := s.API.GetTestTimingManifest(ctx, cfg.SuiteID)
	if err != nil {
		return errors.WithStack(err)
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
	s.Log.Infoln(strings.Join(activePartition.TestFilePaths, cfg.Delimiter))

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
	if err != nil {
		return nil, s.logError(errors.NewSystemError("unable to expand filepath glob: %s", err))
	}
	if len(expandedFilepaths) == 0 {
		s.Log.Debug("No paths to test results provided")
		return nil, nil
	}

	parsedResults, err := s.parse(expandedFilepaths)
	if err != nil {
		return nil, s.logError(errors.WithStack(err))
	}

	return s.performTestResultsUpload(
		ctx,
		testSuiteID,
		parsedResults,
	)
}

func (s Service) performTestResultsUpload(
	ctx context.Context,
	testSuiteID string,
	testResults []v1.TestResults,
) ([]api.TestResultsUploadResult, error) {
	testResultsFiles := make([]api.TestResultsFile, len(testResults))

	for i, testResult := range testResults {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, s.logError(errors.NewInternalError("Unable to generate new UUID: %s", err))
		}
		f, err := os.CreateTemp("", "rwx-test-results")
		if err != nil {
			return nil, s.logError(errors.NewInternalError("Unable to write test results to file: %s", err))
		}
		defer os.Remove(f.Name())

		buf, err := json.Marshal(testResult)
		if err != nil {
			return nil, s.logError(errors.NewInternalError("Unable to output test results as JSON: %s", err))
		}
		if _, err := f.Write(buf); err != nil {
			return nil, s.logError(errors.NewInternalError("Unable to write test results to file: %s", err))
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, s.logError(errors.NewInternalError("Unable to rewind test results file: %s", err))
		}
		defer f.Close()

		originalPaths := make([]string, len(testResult.DerivedFrom))
		for i, originalTestResult := range testResult.DerivedFrom {
			originalPaths[i] = originalTestResult.OriginalFilePath
		}

		testResultsFiles[i] = api.TestResultsFile{
			ExternalID:    id,
			FD:            f,
			OriginalPaths: originalPaths,
			Parser:        api.ParserTypeRWX,
		}
	}

	uploadResults, err := s.API.UploadTestResults(ctx, testSuiteID, testResultsFiles)
	if err != nil {
		return nil, s.logError(errors.WithMessage(err, "unable to upload test result"))
	}

	return uploadResults, nil
}
