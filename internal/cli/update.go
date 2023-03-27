package cli

import (
	"context"
	"strings"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/backend/local"
	"github.com/rwx-research/captain-cli/internal/errors"
)

func parseFlags(args []string) local.Map {
	order := make([]string, 0)
	values := make(map[string]string)

	for i, arg := range args {
		if len(arg) < 2 {
			continue
		}

		if arg[:2] == "--" {
			fields := strings.SplitN(arg[2:], "=", 2)
			if len(fields) > 1 {
				values[fields[0]] = fields[1]
			} else if len(args) > i+1 {
				values[fields[0]] = args[i+1]
			}

			if fields[0] != "suite-id" {
				order = append(order, fields[0])
			}
		}
	}

	return local.Map{Order: order, Values: values}
}

func (s Service) AddFlake(ctx context.Context, args []string) error {
	localStorage, ok := s.API.(local.Client)
	if !ok {
		return errors.NewConfigurationError("captain add flake only works with a local file backend.")
	}

	localStorage.Flakes = append(localStorage.Flakes, parseFlags(args).ToYAML())

	return errors.WithStack(localStorage.Flush())
}

func (s Service) AddQuarantine(ctx context.Context, args []string) error {
	localStorage, ok := s.API.(local.Client)
	if !ok {
		return errors.NewConfigurationError("captain add flake only works with a local file backend.")
	}

	localStorage.Quarantines = append(localStorage.Quarantines, parseFlags(args).ToYAML())

	return errors.WithStack(localStorage.Flush())
}

func (s Service) RemoveFlake(ctx context.Context, args []string) error {
	localStorage, ok := s.API.(local.Client)
	if !ok {
		return errors.NewConfigurationError("captain add flake only works with a local file backend.")
	}

	identity := parseFlags(args)
	for i := len(localStorage.Flakes) - 1; i >= 0; i-- {
		if identity.Equals(local.NewMapFromYAML(localStorage.Flakes[i])) {
			localStorage.Flakes = append(localStorage.Flakes[:i], localStorage.Flakes[i+1:]...)
		}
	}

	return errors.WithStack(localStorage.Flush())
}

func (s Service) RemoveQuarantine(ctx context.Context, args []string) error {
	localStorage, ok := s.API.(local.Client)
	if !ok {
		return errors.NewConfigurationError("captain add flake only works with a local file backend.")
	}

	identity := parseFlags(args)
	for i := len(localStorage.Quarantines) - 1; i >= 0; i-- {
		if identity.Equals(local.NewMapFromYAML(localStorage.Quarantines[i])) {
			localStorage.Quarantines = append(localStorage.Quarantines[:i], localStorage.Quarantines[i+1:]...)
		}
	}

	return errors.WithStack(localStorage.Flush())
}

// UploadTestResults is the implementation of `captain upload results`.
// Deprecated: Use `captain update results` instead, which supports both the local and remote backend.
func (s Service) UploadTestResults(
	ctx context.Context,
	testSuiteID string,
	filepaths []string,
) ([]backend.TestResultsUploadResult, error) {
	// only attempt the upload if the CLI is set up to interact with Captain Cloud.
	if _, ok := s.API.(local.Client); ok {
		return nil, errors.NewConfigurationError("Uploading test results requires a Captain Cloud subscription.")
	}

	return s.UpdateTestResults(ctx, testSuiteID, filepaths)
}

// UpdateTestResults is the implementation of `captain update results`. It takes a number of filepaths, checks that
// files exist and uploads them using the provided API.
func (s Service) UpdateTestResults(
	ctx context.Context,
	testSuiteID string,
	filepaths []string,
) ([]backend.TestResultsUploadResult, error) {
	if testSuiteID == "" {
		return nil, errors.NewConfigurationError("suite-id is required")
	}

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

	parsedResults, err := s.parse(expandedFilepaths, 1)
	if err != nil {
		return nil, s.logError(errors.WithStack(err))
	}

	result, err := s.API.UpdateTestResults(ctx, testSuiteID, *parsedResults)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update test results")
	}

	return result, nil
}
