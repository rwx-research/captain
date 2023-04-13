package cli

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/rwx-research/captain-cli/internal/config"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// RunConfig holds the configuration for running a test suite (used by `RunSuite`)
type RunConfig struct {
	Args                      []string
	Command                   string
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
	UploadResults             bool
	PartitionIndex            int
	PartitionTotal            int
	PartitionGlob             []string
	PartitionDelimeter        string
	PartitionCommandTemplate  string
}

var maxTestsToRetryRegexp = regexp.MustCompile(
	`^\s*(?P<failureCount>\d+)\s*$|^\s*(?:(?P<failurePercentage>\d+(?:\.\d+)?)%)\s*$`,
)

func (rc RunConfig) Validate() error {
	if rc.RetryCommandTemplate == "" && (rc.Retries > 0 || rc.FlakyRetries > 0) {
		return errors.NewConfigurationError(
			"Missing retry command",
			"You seem to have retries enabled, but there is no retry command template configured.",
			"The retry command template can be set using the --retry-command flag. Alternatively, you can "+
				"use the Captain configuration file to permanently set a command template for a "+
				"given test suite.",
		)
	}

	if rc.MaxTestsToRetry != "" && !maxTestsToRetryRegexp.MatchString(rc.MaxTestsToRetry) {
		return errors.NewConfigurationError(
			"Unsupported --max-tests-to-retry value",
			fmt.Sprintf(
				"Captain is unable to parse the --max-tests-to-retry option, which is currently set to %q",
				rc.MaxTestsToRetry,
			),
			"It is expected that this option is either set to a positive integer or to a percentage of the total "+
				"tests to retry. Percentages can be fractional.",
		)
	}

	if rc.PartitionCommandTemplate != "" && (rc.IsRunningPartition()) {
		return errors.NewConfigurationError(
			"Missing partition command",
			"You seem to be passing partition specific options, but there is no partition command template configured.",
			"The partition command template can be set using the --partition-command flag. Alternatively, you can "+
				"use the Captain configuration file to permanently set a partition command template for a "+
				"given test suite.",
		)
	}

	if len(rc.PartitionGlob) == 0 && (rc.IsRunningPartition()) {
		return errors.NewConfigurationError(
			"Missing partition glob configuration",
			"You seem to be passing partition specific options, but have not provided glob(s) of where to locate your test files.",
			"The partition glob can be set using the --partition-glob flag. Alternatively, you can "+
				"use the Captain configuration file to permanently set partition globs for a given test suite.",
		)
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

func (rc RunConfig) IsRunningPartition() bool {
	return rc.PartitionIndex == -1 || rc.PartitionTotal == -1
}

type PartitionConfig struct {
	SuiteID        string
	TestFilePaths  []string
	Delimiter      string
	PartitionNodes config.PartitionNodes
}

func (pc PartitionConfig) Validate() error {
	if pc.SuiteID == "" {
		return errors.NewConfigurationError(
			"Missing suite ID",
			"A suite ID is required in order to use the partitioning feature.",
			"The suite ID can be set using the --suite-id flag or setting a CAPTAIN_SUITE_ID environment variable",
		)
	}

	if pc.PartitionNodes.Total <= 0 {
		return errors.NewConfigurationError(
			"Missing total partition count",
			"In order to use the partitioning feature, Captain needs to know the total number of partitions.",
			"The total number of partitions can be set using the --total flag or alternatively the "+
				"CAPTAIN_PARTITION_TOTAL environment variable.",
		)
	}

	if pc.PartitionNodes.Index < 0 {
		return errors.NewConfigurationError(
			"Missing partition index",
			"Captain is missing the index of the partition that you would like to generate.",
			"The partition index can be set using the --index flag or alternatively the CAPTAIN_PARTITION_INDEX "+
				"environment variable.",
		)
	}

	if pc.PartitionNodes.Index >= pc.PartitionNodes.Total {
		return errors.NewConfigurationError(
			"Unsupported partitioning setup",
			fmt.Sprintf(
				"You specified a partition index (%d) that is greater than or equal to the total number of partitions (%d)",
				pc.PartitionNodes.Index, pc.PartitionNodes.Total,
			),
			"Please set the index to be below the total number of partitions. Note that captain is using '0' as the first "+
				"partition number",
		)
	}

	if len(pc.TestFilePaths) == 0 {
		return errors.NewConfigurationError(
			"Missing test file paths",
			"No test file paths are given.",
			"Please specify the path or paths to your test files as arguments to the 'captain partition' command.\n\n"+
				"\tcaptain partition [flags] <filepath>\n\n"+
				"You can also execute 'captain partition --help' for further information.",
		)
	}

	return nil
}

type QuarantineConfig struct {
	Args                []string
	TestResultsFileGlob string
	PrintSummary        bool
	Quiet               bool
	Reporters           map[string]Reporter
	SuiteID             string
	UpdateStoredResults bool
}
