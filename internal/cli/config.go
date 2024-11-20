package cli

import (
	"fmt"
	"regexp"
	"strconv"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/config"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// RunConfig holds the configuration for running a test suite (used by `RunSuite`)
type RunConfig struct {
	Args                        []string
	CloudOrganizationSlug       string
	Command                     string
	TestResultsFileGlob         string
	FailOnUploadError           bool
	FailRetriesFast             bool
	FlakyRetries                int
	IntermediateArtifactsPath   string
	MaxTestsToRetry             string
	PostRetryCommands           []string
	PreRetryCommands            []string
	PrintSummary                bool
	Quiet                       bool
	Reporters                   map[string]Reporter
	Retries                     int
	RetryCommandTemplate        string
	SuiteID                     string
	SubstitutionsByFramework    map[v1.Framework]targetedretries.Substitution
	UpdateStoredResults         bool
	UploadResults               bool
	PartitionCommandTemplate    string
	PartitionConfig             PartitionConfig
	WriteRetryFailedTestsAction bool
	DidRetryFailedTestsInMint   bool
}

var maxTestsToRetryRegexp = regexp.MustCompile(
	`^\s*(?P<failureCount>\d+)\s*$|^\s*(?:(?P<failurePercentage>\d+(?:\.\d+)?)%)\s*$`,
)

func (rc RunConfig) Validate(log *zap.SugaredLogger) error {
	if rc.RetryCommandTemplate == "" && (rc.Retries > 0 || rc.FlakyRetries > 0) {
		return errors.NewConfigurationError(
			"Missing retry command",
			"You seem to have retries enabled, but there is no retry command template configured.",
			"The retry command template can be set using the --retry-command flag. Alternatively, you can "+
				"use the Captain configuration file to permanently set a command template for a "+
				"given test suite.",
		)
	}

	if rc.RetryCommandTemplate != "" && !(rc.Retries > 0 || rc.FlakyRetries > 0) {
		log.Warn("There is a retry command configured for this test suite, however the retry count is set to 0.")
		log.Warn("Retries are disabled.")
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

	if rc.MaxTestsToRetry != "" && !(rc.Retries > 0 || rc.FlakyRetries > 0) {
		log.Warn("The --max-tests-to-retry flag has no effect as no retries are otherwise configured.")
	}

	if rc.PartitionCommandTemplate != "" && rc.PartitionConfig.PartitionNodes.Total <= 1 {
		log.Warnf("There is a partition command configured for this test suite, but partitioning is disabled.")
	}

	if rc.IsRunningPartition() {
		err := rc.PartitionConfig.Validate()
		if err != nil {
			return errors.WithStack(err)
		}
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
	// TODO: Should we have a bit somewhere that indicates provider defaulted?
	return rc.PartitionCommandTemplate != "" && rc.PartitionConfig.PartitionNodes.Total >= 1
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
			"In order to use the partitioning feature, Captain needs to know the total number of partitions.\n",
			"When using the run command, the total number of partitions can be set using the --partition-total flag.\n\n"+
				"When using the partition command, the total number of partitions can be set using the --total flag or "+
				"alternatively the CAPTAIN_PARTITION_TOTAL environment variable.",
		)
	}

	if pc.PartitionNodes.Index < 0 {
		return errors.NewConfigurationError(
			"Missing partition index",
			"Captain is missing the index of the partition that you would like to generate.\n",
			"When using the run command, partition index can be set using the --partition-index flag.\n\n"+
				"When using the partition command, partition index can be set using the --index flag "+
				"or alternatively the CAPTAIN_PARTITION_INDEX environment variable.",
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
			"No test file paths are provided.\n",
			"When using the run command, please specify the path or paths to your test files using the --partition-globs flag. "+
				"You may specify this flag multiple times if needed.\n\n"+
				"When using the partition command, please specify the path or paths to your test files as arguments.\n\n"+
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
