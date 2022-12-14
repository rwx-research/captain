package parsing

import (
	"encoding/json"
	"io"
	"regexp"

	ginkgo "github.com/onsi/ginkgo/v2/types"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type GoGinkgoParser struct{}

var goGinkgoNewlineRegexp = regexp.MustCompile(`\r?\n`)

func (p GoGinkgoParser) Parse(data io.Reader) (*v1.TestResults, error) {
	var ginkgoReport []ginkgo.Report

	if err := json.NewDecoder(data).Decode(&ginkgoReport); err != nil {
		return nil, errors.NewInputError("Unable to parse test results as JSON: %s", err)
	}
	if len(ginkgoReport) == 0 {
		return nil, errors.NewInputError("Report is empty; unable to guarantee the results are Ginkgo")
	}

	tests := make([]v1.Test, 0)
	otherErrors := make([]v1.OtherError, 0)
	for _, suite := range ginkgoReport {
		if suite.SuitePath == "" {
			return nil, errors.NewInputError("Report does not look like a Ginkgo report")
		}

		for _, reason := range suite.SpecialSuiteFailureReasons {
			location := v1.Location{File: suite.SuitePath}
			otherErrors = append(otherErrors, v1.OtherError{Message: reason, Location: &location})
		}

		for _, specReport := range suite.SpecReports {
			// https://github.com/onsi/ginkgo/blob/60240d1b6479bb8bba394760ec0034b98d5c3f7b/types/types.go#L279-L283
			lineage := make([]string, 0)
			lineage = append(lineage, specReport.ContainerHierarchyTexts...)
			if specReport.LeafNodeText != "" {
				lineage = append(lineage, specReport.LeafNodeText)
			}

			line := specReport.LineNumber()
			location := v1.Location{File: specReport.FileName(), Line: &line}

			var status v1.TestStatus
			switch specReport.State {
			case ginkgo.SpecStateInvalid:
				return nil, errors.NewInputError("Invalid spec state: %v", specReport.State)
			case ginkgo.SpecStatePending:
				status = v1.NewPendedTestStatus(nil)
			case ginkgo.SpecStateSkipped:
				message := specReport.Failure.Message
				status = v1.NewSkippedTestStatus(&message)
			case ginkgo.SpecStatePassed:
				status = v1.NewSuccessfulTestStatus()
			case ginkgo.SpecStateFailed:
				message := specReport.Failure.Message
				backtrace := goGinkgoNewlineRegexp.Split(specReport.Failure.Location.FullStackTrace, -1)
				status = v1.NewFailedTestStatus(&message, nil, backtrace)
			case ginkgo.SpecStateAborted:
				status = v1.NewCanceledTestStatus()
			case ginkgo.SpecStatePanicked:
				message := "The test panicked"
				status = v1.NewFailedTestStatus(&message, nil, nil)
			case ginkgo.SpecStateInterrupted:
				status = v1.NewCanceledTestStatus()
			case ginkgo.SpecStateTimedout:
				status = v1.NewTimedOutTestStatus()
			default:
				return nil, errors.NewInputError("Unknown spec state: %v", specReport.State)
			}

			duration := specReport.RunTime
			stderr := specReport.CapturedStdOutErr
			stdout := specReport.CapturedGinkgoWriterOutput
			startedAt := specReport.StartTime
			finishedAt := specReport.EndTime
			pastAttempts := make([]v1.TestAttempt, len(specReport.AdditionalFailures))
			for i, failure := range specReport.AdditionalFailures {
				message := failure.Failure.Message
				backtrace := goGinkgoNewlineRegexp.Split(failure.Failure.Location.FullStackTrace, -1)
				pastAttempts[i] = v1.TestAttempt{
					Status: v1.NewFailedTestStatus(&message, nil, backtrace),
				}
			}

			tests = append(tests, v1.Test{
				Name:     specReport.FullText(),
				Lineage:  lineage,
				Location: &location,
				Attempt: v1.TestAttempt{
					Duration:   &duration,
					Status:     status,
					Meta:       map[string]any{"labels": specReport.Labels()},
					Stderr:     &stderr,
					Stdout:     &stdout,
					StartedAt:  &startedAt,
					FinishedAt: &finishedAt,
				},
				PastAttempts: pastAttempts,
			})
		}
	}

	return v1.NewTestResults(
		v1.GoGinkgoFramework,
		tests,
		otherErrors,
	), nil
}
