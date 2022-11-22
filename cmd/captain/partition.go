package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
)

var (
	partitionIndex  int
	totalPartitions int
	partitionCmd    = &cobra.Command{
		Use:     "partition",
		Short:   "Partition a test suite using historical file timings recorded by Captain",
		Long:    descriptionPartition,
		PreRunE: initCLIService,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := captain.Partition(cmd.Context(), cli.PartitionConfig{
				PartitionIndex:  partitionIndex,
				SuiteID:         suiteID,
				TestFilePaths:   args,
				TotalPartitions: totalPartitions,
			})
			return errors.Wrap(err)
		},
	}
)

func init() {
	// Although `suite-id` is a global flag, we need to re-define it here in order to mark it as required.
	// This is due to a bug in 'spf13/cobra'. See https://github.com/spf13/cobra/issues/921
	partitionCmd.Flags().StringVar(&suiteID, "suite-id", "", "the id of the test suite (required)")
	partitionCmd.Flags().IntVar(&partitionIndex, "index", 0, "the index of a particular partition (required)")
	partitionCmd.Flags().IntVar(&totalPartitions, "total", 0, "the total number of partitions (required)")

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
