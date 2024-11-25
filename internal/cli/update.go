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

func (s Service) AddFlake(_ context.Context, args []string) error {
	localStorage, ok := s.API.(local.Client)
	if !ok {
		return errors.NewConfigurationError(
			"'captain add flake' only works in OSS mode",
			"You are trying to register a new flake in Captain, however it appears that you are using "+
				"Captain Cloud.",
			"Please visit https://cloud.rwx.com/captain to configure your flakes or quarantines.",
		)
	}

	localStorage.Flakes = append(localStorage.Flakes, parseFlags(args).ToYAML())

	return errors.WithStack(localStorage.Flush())
}

func (s Service) AddQuarantine(_ context.Context, args []string) error {
	localStorage, ok := s.API.(local.Client)
	if !ok {
		return errors.NewConfigurationError(
			"'captain add quarantine' only works in OSS mode",
			"You are trying to quarantine a new test in Captain, however it appears that you are using "+
				"Captain Cloud.",
			"Please visit https://cloud.rwx.com/captain to configure your flakes or quarantines.",
		)
	}

	localStorage.Quarantines = append(localStorage.Quarantines, parseFlags(args).ToYAML())

	return errors.WithStack(localStorage.Flush())
}

func (s Service) RemoveFlake(_ context.Context, args []string) error {
	localStorage, ok := s.API.(local.Client)
	if !ok {
		return errors.NewConfigurationError(
			"'captain remove flake' only works in OSS mode",
			"You are trying to remove a flake in Captain, however it appears that you are using "+
				"Captain Cloud.",
			"Please visit https://cloud.rwx.com/captain to configure your flakes or quarantines.",
		)
	}

	identity := parseFlags(args)
	for i := len(localStorage.Flakes) - 1; i >= 0; i-- {
		if identity.Equals(local.NewMapFromYAML(localStorage.Flakes[i])) {
			localStorage.Flakes = append(localStorage.Flakes[:i], localStorage.Flakes[i+1:]...)
		}
	}

	return errors.WithStack(localStorage.Flush())
}

func (s Service) RemoveQuarantine(_ context.Context, args []string) error {
	localStorage, ok := s.API.(local.Client)
	if !ok {
		return errors.NewConfigurationError(
			"'captain remove quarantine' only works in OSS mode",
			"You are trying to remove a quarantine in Captain, however it appears that you are using "+
				"Captain Cloud.",
			"Please visit https://cloud.rwx.com/captain to configure your flakes or quarantines.",
		)
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
		return nil, errors.NewConfigurationError(
			"Missing Cloud subscription",
			"Uploading test results requires a Captain Cloud subscription.",
			"Please visit https://www.rwx.com to learn about purchasing options.",
		)
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
		return nil, errors.NewConfigurationError(
			"Missing suite ID",
			"A suite ID is required in order to use the upload feature.",
			"The suite ID can be set using the --suite-id flag or setting a CAPTAIN_SUITE_ID environment variable.",
		)
	}

	if len(filepaths) == 0 {
		s.Log.Debug("No paths to test results provided")
		return nil, nil
	}

	expandedFilepaths, err := s.FileSystem.GlobMany(filepaths)
	if err != nil {
		return nil, errors.NewSystemError("unable to expand filepath glob: %s", err)
	}
	if len(expandedFilepaths) == 0 {
		s.Log.Debug("No paths to test results provided")
		return nil, nil
	}

	parsedResults, err := s.parse(expandedFilepaths, 1)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result, err := s.API.UpdateTestResults(ctx, testSuiteID, *parsedResults, true)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update test results")
	}

	return result, nil
}
