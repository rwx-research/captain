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
	Args                     []string
	TestResultsFileGlob      string
	FailOnUploadError        bool
	FlakyRetries             int
	PostRetryCommands        []string
	PreRetryCommands         []string
	PrintSummary             bool
	Quiet                    bool
	Reporters                map[string]Reporter
	Retries                  int
	RetryCommandTemplate     string
	RetryFailureLimit        string
	SuiteID                  string
	SubstitutionsByFramework map[v1.Framework]targetedretries.Substitution
}

var retryFailureLimitRegexp = regexp.MustCompile(
	`^\s*(?P<failureCount>\d+)\s*$|^\s*(?:(?P<failurePercentage>\d+(?:\.\d+)?)%)\s*$`,
)

func (rc RunConfig) Validate() error {
	if rc.RetryCommandTemplate == "" && (rc.Retries > 0 || rc.FlakyRetries > 0) {
		return errors.NewConfigurationError("retry-command must be provided if retries or flaky-retries are > 0")
	}

	if rc.RetryFailureLimit != "" && !retryFailureLimitRegexp.MatchString(rc.RetryFailureLimit) {
		return errors.NewConfigurationError("retry-failure-limit must be either an integer or percentage")
	}

	return nil
}

func (rc RunConfig) RetryFailureLimitCount() (*int, error) {
	if rc.RetryFailureLimit == "" {
		return nil, nil
	}

	match := retryFailureLimitRegexp.FindStringSubmatch(rc.RetryFailureLimit)
	result := map[string]string{}
	for i, name := range retryFailureLimitRegexp.SubexpNames() {
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

func (rc RunConfig) RetryFailureLimitPercentage() (*float64, error) {
	if rc.RetryFailureLimit == "" {
		return nil, nil
	}

	match := retryFailureLimitRegexp.FindStringSubmatch(rc.RetryFailureLimit)
	result := map[string]string{}
	for i, name := range retryFailureLimitRegexp.SubexpNames() {
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

type PartitionConfig struct {
	PartitionIndex  int
	SuiteID         string
	TestFilePaths   []string
	TotalPartitions int
	Delimiter       string
}

func (pc PartitionConfig) Validate() error {
	if pc.TotalPartitions <= 0 {
		return errors.NewConfigurationError("total must be > 0")
	}

	if pc.PartitionIndex < 0 {
		return errors.NewConfigurationError("index must be >= 0")
	}

	if pc.PartitionIndex >= pc.TotalPartitions {
		return errors.NewConfigurationError("index must be < total")
	}

	if len(pc.TestFilePaths) == 0 {
		return errors.NewConfigurationError("no test file paths provided")
	}

	return nil
}
