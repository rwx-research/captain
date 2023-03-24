package cli

import (
	"regexp"
	"strconv"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// RunConfig holds the configuration for running a test suite (used by `RunSuite`)
type RunConfig struct {
	Args                      []string
	TestResultsFileGlob       string
	FailOnUploadError         bool
	FailRetriesFast           bool
	FlakyRetries              int
	IntermediateArtifactsPath string
	MaxTestsToRetry           string
	PostRetryCommands         []string
	PreRetryCommands          []string
	PrintSummary              bool
	Quiet                     bool
	Reporters                 map[string]Reporter
	Retries                   int
	RetryCommandTemplate      string
	SuiteID                   string
	SubstitutionsByFramework  map[v1.Framework]targetedretries.Substitution
	UpdateStoredResults       bool
}

var maxTestsToRetryRegexp = regexp.MustCompile(
	`^\s*(?P<failureCount>\d+)\s*$|^\s*(?:(?P<failurePercentage>\d+(?:\.\d+)?)%)\s*$`,
)

func (rc RunConfig) Validate() error {
	if rc.RetryCommandTemplate == "" && (rc.Retries > 0 || rc.FlakyRetries > 0) {
		return errors.NewConfigurationError("retry-command must be provided if retries or flaky-retries are > 0")
	}

	if rc.MaxTestsToRetry != "" && !maxTestsToRetryRegexp.MatchString(rc.MaxTestsToRetry) {
		return errors.NewConfigurationError("max-tests-to-retry must be either an integer or percentage")
	}

	return nil
}

func (rc RunConfig) MaxTestsToRetryCount() (*int, error) {
	if rc.MaxTestsToRetry == "" {
		return nil, nil
	}

	match := maxTestsToRetryRegexp.FindStringSubmatch(rc.MaxTestsToRetry)
	result := map[string]string{}
	for i, name := range maxTestsToRetryRegexp.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	value := result["failureCount"]
	if value == "" {
		return nil, nil
	}

	count, err := strconv.Atoi(value)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &count, nil
}

func (rc RunConfig) MaxTestsToRetryPercentage() (*float64, error) {
	if rc.MaxTestsToRetry == "" {
		return nil, nil
	}

	match := maxTestsToRetryRegexp.FindStringSubmatch(rc.MaxTestsToRetry)
	result := map[string]string{}
	for i, name := range maxTestsToRetryRegexp.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	value := result["failurePercentage"]
	if value == "" {
		return nil, nil
	}

	percentage, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &percentage, nil
}

type PartitionNodes struct {
	Total int
	Index int
}

type PartitionConfig struct {
	SuiteID        string
	TestFilePaths  []string
	Delimiter      string
	PartitionNodes PartitionNodes
}

func (pc PartitionConfig) Validate() error {
	if pc.SuiteID == "" {
		return errors.NewConfigurationError("suite-id is required")
	}

	if pc.PartitionNodes.Total <= 0 {
		return errors.NewConfigurationError("total must be > 0")
	}

	if pc.PartitionNodes.Index < 0 {
		return errors.NewConfigurationError("index must be >= 0")
	}

	if pc.PartitionNodes.Index >= pc.PartitionNodes.Total {
		return errors.NewConfigurationError("index must be < total")
	}

	if len(pc.TestFilePaths) == 0 {
		return errors.NewConfigurationError("no test file paths provided")
	}

	return nil
}
