package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-shellwords"
	"golang.org/x/sync/errgroup"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/backend/local"
	"github.com/rwx-research/captain-cli/internal/backend/remote"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/providers"
	"github.com/rwx-research/captain-cli/internal/reporting"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// RunSuite runs the specified build- or test-suite and optionally uploads the resulting test results file.
func (s Service) RunSuite(ctx context.Context, cfg RunConfig) (finalErr error) {
	err := cfg.Validate()
	if err != nil {
		return errors.WithStack(err)
	}

	// Fetch run configuration in the background
	var apiConfiguration backend.RunConfiguration
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		apiConfiguration, err = s.API.GetRunConfiguration(egCtx, cfg.SuiteID)
		if err != nil {
			return errors.WithStack(err)
		}

		if len(apiConfiguration.QuarantinedTests) == 0 {
			s.Log.Debug("No quarantined tests defined in Captain")
		}

		return nil
	})

	stdout := os.Stdout
	if cfg.Quiet {
		// According to the documentation, passing in a nil pointer to `os.Exec`
		// should also work - however, that seems to open /dev/null with the
		// wrong flags, leading to errors like
		// `cat: standard output: Bad file descriptor`
		stdout, err = os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, 0o666)
		if err != nil {
			s.Log.Warnf("Could not open %s for writing", os.DevNull)
		}
	}

	runCommand, err := s.MakeRunCommand(ctx, cfg)
	if err != nil {
		return errors.Wrapf(err, "Failed to assemble run command")
	}

	// Short circuit and print warning info (e.g attempting to run an empty partition)
	if runCommand.shortCircuit {
		s.Log.Warnf(runCommand.shortCircuitInfo)
		os.Exit(0)
	}

	commandArgs, err := runCommand.CommandArgs()
	if err != nil {
		return errors.Wrapf(err, "Failed to assemble run command args")
	}

	// Run sub-command
	ctx, cmdErr := s.runCommand(ctx, commandArgs, stdout, true)
	defer func() {
		if abqErr := s.setAbqExitCode(ctx, finalErr); abqErr != nil {
			finalErr = errors.Wrap(finalErr, abqErr.Error())
			if finalErr == nil {
				finalErr = abqErr
			}
			s.Log.Errorf("Error setting ABQ exit code: %v", finalErr)
		}
	}()
	testResults, testResultsFiles, runErr, err := s.handleCommandOutcome(cfg, cmdErr, 1)
	if err != nil {
		return err
	}

	// Wait until run configuration was fetched. Ignore any errors.
	if err := eg.Wait(); err != nil {
		s.Log.Warnf("Unable to fetch run configuration from Captain: %s", err)
	}

	testResults, didRetry, err := s.attemptRetries(ctx, testResults, testResultsFiles, cfg, apiConfiguration)
	if err != nil {
		s.Log.Warnf("An issue occurred while retrying your tests: %v", err)
	}

	quarantinedFailedTests := make([]v1.Test, 0)
	unquarantinedFailedTests := make([]v1.Test, 0)
	otherErrorCount := 0

	if testResults != nil {
		otherErrorCount = testResults.Summary.OtherErrors

		quarantinedTests := make([]backend.Test, len(apiConfiguration.QuarantinedTests))
		for i, quarantinedTest := range apiConfiguration.QuarantinedTests {
			quarantinedTests[i] = quarantinedTest.Test
		}

		for i, test := range testResults.Tests {
			if s.isIdentifiedIn(test, quarantinedTests) && test.Attempt.Status.PotentiallyFlaky() {
				testResults.Tests[i] = test.Quarantine()
				s.Log.Debugf("quarantined %v test: %v", test.Attempt.Status, test)
				quarantinedFailedTests = append(quarantinedFailedTests, test)
			} else if test.Attempt.Status.ImpliesFailure() {
				s.Log.Debugf("did not quarantine %v test: %v", test.Attempt.Status, test)
				unquarantinedFailedTests = append(unquarantinedFailedTests, test)
			}
		}
		testResults.Summary = v1.NewSummary(testResults.Tests, testResults.OtherErrors)
	}

	var uploadResults []backend.TestResultsUploadResult
	var uploadError error
	var headerPrinted bool

	if cfg.PrintSummary {
		s.printHeader()
		headerPrinted = true
	}

	// We ignore the error here since `UploadTestResults` will already log any errors. Furthermore, any errors here will
	// not affect the exit code.
	if testResults != nil {
		uploadResults, uploadError = s.reportTestResults(ctx, cfg, *testResults)
	} else {
		s.Log.Debugf("No test results were parsed. Globbed files: %v", testResultsFiles)
	}

	// Display detailed output if necessary
	hasUploadResults := len(uploadResults) > 0
	hasQuarantinedFailedTests := len(quarantinedFailedTests) > 0
	hasDetails := hasUploadResults || hasQuarantinedFailedTests

	if hasDetails && !headerPrinted && !cfg.Quiet {
		s.printHeader()
		headerPrinted = true
	}

	if hasUploadResults && headerPrinted {
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
			s.Log.Infoln(fmt.Sprintf("- Updated Captain with results from %v", uploadedPath))
		}
		for _, erroredPath := range erroredPaths {
			s.Log.Infoln(fmt.Sprintf("- Unable to update Captain with results from %v", erroredPath))
		}
	}

	// This section is always printed, even when `--quiet` is specified
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

	// Return the original exit code if there was a non-test error
	if runErr != nil && otherErrorCount > 0 {
		return errors.WithStack(runErr)
	}

	// Return the original exit code if we didn't detect any failures
	// (perhaps an error in parsing, failure of the framework to output an artifact, or coverage requirement)
	if runErr != nil && len(quarantinedFailedTests)+len(unquarantinedFailedTests) == 0 && !didRetry {
		return errors.WithStack(runErr)
	}

	// Return the original exit code if we didn't quarantine all of the failures we saw
	if runErr != nil && len(unquarantinedFailedTests) > 0 {
		return errors.WithStack(runErr)
	}

	if uploadError != nil && cfg.FailOnUploadError {
		return uploadError
	}

	return nil
}

func (s Service) attemptRetries(
	ctx context.Context,
	originalTestResults *v1.TestResults,
	originalTestResultsFiles []string,
	cfg RunConfig,
	apiConfiguration backend.RunConfiguration,
) (*v1.TestResults, bool, error) {
	nonFlakyRetries := cfg.Retries
	flakyRetries := cfg.FlakyRetries

	if nonFlakyRetries <= 0 && flakyRetries <= 0 {
		return originalTestResults, false, nil
	}

	ias, err := s.newIntermediateArtifactStorage(cfg.IntermediateArtifactsPath)
	if err != nil {
		return originalTestResults, false, errors.WithStack(err)
	}

	if cfg.IntermediateArtifactsPath == "" {
		defer func() {
			if err := ias.delete(); err != nil {
				s.Log.Warnf("Unable to clean up temporary files: %s", err.Error())
			}
		}()
	}

	// if retries is set and flaky-retries is not, set flaky-retries to retries
	// this way, we can isolate the logic for flaky and non-flaky instead of special casing everywhere
	// note: it's important we don't do this the other way around; retries implies flaky-retries, flaky-retries
	// does not imply retries
	if nonFlakyRetries > 0 && flakyRetries < 0 {
		flakyRetries = nonFlakyRetries
	}

	if didRun, err := s.didAbqRun(ctx); didRun || err != nil {
		if err != nil {
			return originalTestResults, false, errors.WithStack(err)
		}

		return originalTestResults, false, errors.NewInputError("Captain retries cannot be used with ABQ")
	}

	if originalTestResults == nil {
		return originalTestResults, false, errors.NewInternalError("No test results detected")
	}

	compiledTemplate, err := templating.CompileTemplate(cfg.RetryCommandTemplate)
	if err != nil {
		return originalTestResults, false, errors.WithStack(err)
	}

	framework := originalTestResults.Framework
	var substitution targetedretries.Substitution = targetedretries.JSONSubstitution{FileSystem: s.FileSystem}
	if err := substitution.ValidateTemplate(compiledTemplate); err != nil {
		frameworkSubstitution, ok := cfg.SubstitutionsByFramework[framework]
		if !ok {
			return originalTestResults, false, errors.NewInternalError("Unable to retry %q", framework)
		}

		if err := frameworkSubstitution.ValidateTemplate(compiledTemplate); err != nil {
			return originalTestResults, false, errors.WithStack(err)
		}

		substitution = frameworkSubstitution
	}

	flattenedTestResults := originalTestResults

	if err := ias.moveTestResults(originalTestResultsFiles); err != nil {
		return flattenedTestResults, false, errors.WithStack(err)
	}

	maxTestsToRetryCount, err := cfg.MaxTestsToRetryCount()
	if err != nil {
		return flattenedTestResults, false, errors.WithStack(err)
	}

	maxTestsToRetryPercentage, err := cfg.MaxTestsToRetryPercentage()
	if err != nil {
		return flattenedTestResults, false, errors.WithStack(err)
	}

	maxRetries := nonFlakyRetries
	if flakyRetries > maxRetries {
		maxRetries = flakyRetries
	}
	formattedRetryTotal := fmt.Sprintf(" of %v", maxRetries)
	if flakyRetries > 0 && nonFlakyRetries > 0 && nonFlakyRetries != flakyRetries {
		formattedRetryTotal = ""
	}

	for retries := 0; retries < maxRetries; retries++ {
		remainingFlakyFailures := make([]v1.Test, 0)
		remainingNonFlakyFailures := make([]v1.Test, 0)

		ias.setRetryID(retries + 1)

		for _, test := range flattenedTestResults.Tests {
			if !test.Attempt.Status.ImpliesFailure() {
				continue
			}

			if s.isIdentifiedIn(test, apiConfiguration.FlakyTests) {
				remainingFlakyFailures = append(remainingFlakyFailures, test)
			} else {
				remainingNonFlakyFailures = append(remainingNonFlakyFailures, test)
			}
		}

		nonFlakyAttemptsExhausted := retries >= nonFlakyRetries
		flakyAttemptsExhausted := retries >= flakyRetries

		testsRemaining := 0
		if !nonFlakyAttemptsExhausted {
			testsRemaining += len(remainingNonFlakyFailures)
		}
		if !flakyAttemptsExhausted {
			testsRemaining += len(remainingFlakyFailures)
		}

		// bail early if there are too many failed tests
		if maxTestsToRetryCount != nil && testsRemaining > *maxTestsToRetryCount {
			break
		}

		// bail early if there are too many failed tests
		testCount := float64(flattenedTestResults.Summary.Tests)
		if maxTestsToRetryPercentage != nil &&
			float64(testsRemaining) > testCount**maxTestsToRetryPercentage/100 {
			break
		}

		// nothing left to retry
		if testsRemaining == 0 {
			break
		}

		// all attempts exhausted
		if nonFlakyAttemptsExhausted && flakyAttemptsExhausted {
			break
		}

		// fail fast if we know we can't pass the build
		if cfg.FailRetriesFast && ((nonFlakyAttemptsExhausted && len(remainingNonFlakyFailures) > 0) ||
			(flakyAttemptsExhausted && len(remainingFlakyFailures) > 0)) {
			break
		}

		filter := func(test v1.Test) bool {
			testIsFlaky := false
			for _, remainingFlakyFailure := range remainingFlakyFailures {
				if test.Matches(remainingFlakyFailure) {
					testIsFlaky = true
					break
				}
			}

			if retries >= flakyRetries && testIsFlaky {
				s.Log.Debugf("Skipping %v; flaky attempts exhausted\n", test)
				return false
			}

			if retries >= nonFlakyRetries && !testIsFlaky {
				s.Log.Debugf("Skipping %v; non-flaky attempts exhausted\n", test)
				return false
			}

			return true
		}

		allNewTestResults := make([]v1.TestResults, 0)
		allSubstitutions, err := substitution.SubstitutionsFor(compiledTemplate, *flattenedTestResults, filter)
		if err != nil {
			return flattenedTestResults, true, errors.Wrap(err, "Unable construct retry substitutions")
		}
		for i, substitutions := range allSubstitutions {
			command := compiledTemplate.Substitute(substitutions)
			args, err := shellwords.Parse(command)
			if err != nil {
				return flattenedTestResults, true, errors.Wrapf(err, "Unable to parse %q into shell arguments", command)
			}

			ias.setCommandID(i + 1)

			s.Log.Infoln()
			s.Log.Infoln(strings.Repeat("-", 80))
			if len(allSubstitutions) == 1 {
				s.Log.Infoln(fmt.Sprintf("- Retry %v%v", retries+1, formattedRetryTotal))
			} else {
				s.Log.Infoln(fmt.Sprintf(
					"- Retry %v%v, command %v of %v",
					retries+1,
					formattedRetryTotal,
					i+1,
					len(allSubstitutions),
				))
			}
			for keyword, value := range substitutions {
				s.Log.Infoln(fmt.Sprintf("-   %v: %v", keyword, value))
			}
			s.Log.Infoln(strings.Repeat("-", 80))
			s.Log.Infoln()

			stdout := os.Stdout
			if cfg.Quiet {
				stdout, err = os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, 0o666)
				if err != nil {
					s.Log.Warnf("Could not open %s for writing", os.DevNull)
				}
			}

			for _, preRetryCommand := range cfg.PreRetryCommands {
				preRetryArgs, err := shellwords.Parse(preRetryCommand)
				if err != nil {
					return flattenedTestResults, true, errors.Wrapf(err, "Unable to parse %q into shell arguments", preRetryCommand)
				}

				if _, err := s.runCommand(ctx, preRetryArgs, stdout, false); err != nil {
					return flattenedTestResults, true, errors.Wrapf(err, "Error while executing %q", preRetryCommand)
				}
			}

			_, cmdErr := s.runCommand(ctx, args, stdout, false)

			for _, postRetryCommand := range cfg.PostRetryCommands {
				postRetryArgs, err := shellwords.Parse(postRetryCommand)
				if err != nil {
					return flattenedTestResults, true, errors.Wrapf(err, "Unable to parse %q into shell arguments", postRetryCommand)
				}

				if _, err := s.runCommand(ctx, postRetryArgs, stdout, false); err != nil {
					return flattenedTestResults, true, errors.Wrapf(err, "Error while executing %q", postRetryCommand)
				}
			}

			// +1 because it's 1-indexed, +1 because the original attempt was #1
			newTestResults, newTestResultsFiles, _, err := s.handleCommandOutcome(cfg, cmdErr, retries+2)
			if err != nil {
				return flattenedTestResults, true, err
			}

			if newTestResults != nil {
				allNewTestResults = append(allNewTestResults, *newTestResults)
			}
			if err := ias.moveTestResults(newTestResultsFiles); err != nil {
				return flattenedTestResults, true, errors.WithStack(err)
			}
		}
		if jsonSubstitution, ok := substitution.(targetedretries.JSONSubstitution); ok {
			if err := jsonSubstitution.CleanUp(allSubstitutions); err != nil {
				s.Log.Warn(err)
			}
		}
		mergedTestResults := v1.Merge([]v1.TestResults{*flattenedTestResults}, allNewTestResults)
		flattenedTestResults = &mergedTestResults
	}

	s.Log.Debugf("Retries complete, summary: %v\n", flattenedTestResults.Summary)
	return flattenedTestResults, true, nil
}

func (s Service) handleCommandOutcome(
	cfg RunConfig,
	cmdErr error,
	groupNumber int,
) (*v1.TestResults, []string, error, error) {
	var runErr error
	ok := true
	if cmdErr != nil {
		runErr, ok = errors.AsExecutionError(cmdErr)
	}
	if !ok {
		return nil, nil, errors.WithStack(runErr), errors.WithStack(cmdErr)
	}

	// Return early if no testResultsFiles were defined over the CLI. If there was an error
	// during execution, the exit Code is being passed along.
	if cfg.TestResultsFileGlob == "" {
		s.Log.Debug("No testResultsFile path provided, quitting")
		return nil, nil, errors.WithStack(runErr), errors.WithStack(runErr)
	}

	testResultsFiles, err := s.FileSystem.Glob(cfg.TestResultsFileGlob)
	if err != nil {
		return nil,
			testResultsFiles,
			errors.WithStack(runErr),
			errors.NewSystemError("unable to expand filepath glob: %s", err)
	}

	// Parse testResultsFiles
	testResults, err := s.parse(testResultsFiles, groupNumber)
	if err != nil {
		return nil,
			testResultsFiles,
			errors.WithStack(runErr),
			errors.WithStack(err)
	}

	return testResults, testResultsFiles, errors.WithStack(runErr), nil
}

func (s Service) runCommand(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	setAbqEnviron bool,
) (context.Context, error) {
	var environ []string

	if setAbqEnviron {
		ctx, environ = s.applyAbqEnvironment(ctx)
	}

	cmd, err := s.TaskRunner.NewCommand(ctx, exec.CommandConfig{
		Name:   args[0],
		Args:   args[1:],
		Env:    environ,
		Stdout: stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return ctx, errors.NewSystemError("unable to spawn sub-process: %s", err)
	}

	s.Log.Debugf("Executing %q", strings.Join(args, " "))
	if err := cmd.Start(); err != nil {
		return ctx, errors.NewSystemError("unable to execute sub-command: %s", err)
	}
	defer s.Log.Debugf("Finished executing %q", strings.Join(args, " "))

	if err := cmd.Wait(); err != nil {
		if code, e := s.TaskRunner.GetExitStatusFromError(err); e == nil {
			return ctx, errors.NewExecutionError(code, "test suite exited with non-zero exit code")
		}

		return ctx, errors.NewSystemError("Error during program execution: %s", err)
	}

	return ctx, nil
}

func (s Service) isIdentifiedIn(test v1.Test, identifiedTests []backend.Test) bool {
	for _, identifiedTest := range identifiedTests {
		compositeIdentifier, err := test.Identify(
			identifiedTest.IdentityComponents,
			identifiedTest.StrictIdentity,
		)
		if err != nil {
			s.Log.Debugf("%v does not identify %v because %v", identifiedTest, test, err.Error())
			continue
		}

		if compositeIdentifier != identifiedTest.CompositeIdentifier {
			s.Log.Debugf(
				"%v does not identify %v because they have different composite identifiers (%v != %v)",
				identifiedTest,
				test,
				identifiedTest.CompositeIdentifier,
				compositeIdentifier,
			)
			continue
		}

		s.Log.Debugf(
			"%v identifies %v because they share the same composite identifier (%v)",
			identifiedTest,
			test,
			compositeIdentifier,
		)
		return true
	}

	return false
}

func (s Service) reportTestResults(
	ctx context.Context,
	cfg RunConfig,
	testResults v1.TestResults,
) ([]backend.TestResultsUploadResult, error) {
	reportingConfiguration := reporting.Configuration{
		SuiteID:              cfg.SuiteID,
		RetryCommandTemplate: cfg.RetryCommandTemplate,
	}

	if remoteClient, ok := s.API.(remote.Client); ok {
		reportingConfiguration.CloudEnabled = true
		reportingConfiguration.CloudHost = remoteClient.Host
		reportingConfiguration.Provider = remoteClient.Provider
	}

	if _, ok := s.API.(local.Client); ok {
		reportingConfiguration.CloudEnabled = false
		reportingConfiguration.CloudHost = ""
		reportingConfiguration.Provider = providers.Provider{}
	}

	for outputPath, writeReport := range cfg.Reporters {
		file, err := s.FileSystem.Create(outputPath)
		if err == nil {
			err = writeReport(file, testResults, reportingConfiguration)
		}
		if err != nil {
			s.Log.Warnf("Unable to write report to %s: %s", outputPath, err.Error())
		}
	}

	if cfg.PrintSummary {
		if err := reporting.WriteTextSummary(os.Stdout, testResults, reportingConfiguration); err != nil {
			s.Log.Warnf("Unable to write text summary to stdout: %s", err.Error())
		} else {
			// Append an empty line to make output more readable
			fmt.Fprintf(os.Stdout, "\n")
		}
	}

	if _, ok := s.API.(local.Client); ok && !cfg.UpdateStoredResults {
		return nil, nil
	}

	if _, ok := s.API.(remote.Client); ok && !cfg.UploadResults {
		return nil, nil
	}

	result, err := s.API.UpdateTestResults(ctx, cfg.SuiteID, testResults)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update test results")
	}

	return result, nil
}

func (s Service) printHeader() {
	s.Log.Infoln(strings.Repeat("-", 80))
	s.Log.Infoln(fmt.Sprintf("%v Captain %v", strings.Repeat("-", 40-4-1), strings.Repeat("-", 40-3-1)))
	s.Log.Infoln(strings.Repeat("-", 80))
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}
