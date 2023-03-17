package main

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
)

var (
	partitionIndex  int
	totalPartitions int
	delimiter       string
	partitionCmd    = &cobra.Command{
		Use:     "partition",
		Short:   "Partition a test suite using historical file timings recorded by Captain",
		Long:    descriptionPartition,
		PreRunE: initCLIService,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := captain.Partition(cmd.Context(), cli.PartitionConfig{
				SuiteID:       suiteID,
				TestFilePaths: args,
				PartitionNodes: cli.PartitionNodes{
					Index: partitionIndex,
					Total: totalPartitions,
				},
				Delimiter: delimiter,
			})
			return errors.WithStack(err)
		},
	}
)

func init() {
	getEnvIntWithDefault := func(envVar string, defaultValue int) int {
		envInt, err := strconv.Atoi(os.Getenv(envVar))
		if err != nil {
			envInt = defaultValue
		}
		return envInt
	}
	// Although `suite-id` is a global flag, we need to re-define it here in order to mark it as required.
	// This is due to a bug in 'spf13/cobra'. See https://github.com/spf13/cobra/issues/921
	partitionCmd.Flags().StringVar(&suiteID, "suite-id", os.Getenv("CAPTAIN_SUITE_ID"),
		"the id of the test suite (required). Also set with environment variable CAPTAIN_SUITE_ID")
	partitionCmd.Flags().IntVar(&partitionIndex, "index", getEnvIntWithDefault("CAPTAIN_PARTITION_INDEX", 0),
		"the index of a particular partition (required)")
	partitionCmd.Flags().IntVar(&totalPartitions, "total", getEnvIntWithDefault("CAPTAIN_PARTITION_TOTAL", 0),
		"the total number of partitions (required)")
	partitionCmd.Flags().StringVar(&delimiter, "delimiter", " ", "the delimiter used to separate partitioned files")

	if err := partitionCmd.MarkFlagRequired("suite-id"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	if err := partitionCmd.MarkFlagRequired("index"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}

	if err := partitionCmd.MarkFlagRequired("total"); err != nil {
		initializationErrors = append(initializationErrors, err)
	}
	rootCmd.AddCommand(partitionCmd)
}
