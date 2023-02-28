package cli

import (
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// RunConfig holds the configuration for running a test suite (used by `RunSuite`)
type RunConfig struct {
	Args                     []string
	TestResultsFileGlob      string
	FailOnUploadError        bool
	PostRetryCommands        []string
	PreRetryCommands         []string
	PrintSummary             bool
	Quiet                    bool
	Reporters                map[string]Reporter
	Retries                  int
	RetryCommandTemplate     string
	SuiteID                  string
	SubstitutionsByFramework map[v1.Framework]targetedretries.Substitution
}

func (rc RunConfig) Validate() error {
	if rc.Retries < 0 {
		return errors.NewConfigurationError("retries must be >= 0")
	}

	if rc.RetryCommandTemplate == "" && rc.Retries > 0 {
		return errors.NewConfigurationError("retry-command must be provided if retries are > 0")
	}

	return nil
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
