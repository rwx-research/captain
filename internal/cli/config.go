package cli

import "github.com/rwx-research/captain-cli/internal/errors"

// RunConfig holds the configuration for running a test suite (used by `RunSuite`)
type RunConfig struct {
	Args                []string
	TestResultsFileGlob string
	FailOnUploadError   bool
	SuiteID             string
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
