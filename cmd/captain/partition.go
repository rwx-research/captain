package main

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

type partitionArgs struct {
	nodes     cli.PartitionNodes
	delimiter string
}

var (
	pArgs        partitionArgs
	partitionCmd = &cobra.Command{
		Use:   "partition",
		Short: "Partition a test suite using historical file timings recorded by Captain",
		Long:  descriptionPartition,
		PreRunE: initCLIService(func(p providers.Provider) error {
			if p.CommitSha == "" {
				return errors.NewConfigurationError("missing commit sha")
			}
			return nil
		}),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := captain.Partition(cmd.Context(), cli.PartitionConfig{
				SuiteID:        suiteID,
				TestFilePaths:  args,
				PartitionNodes: pArgs.nodes,
				Delimiter:      pArgs.delimiter,
			})
			return errors.WithStack(err)
		},
	}
)

func init() {
	getEnvIntWithDefault := func(envVar string, defaultValue int) int {
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return defaultValue
		}

		envInt, err := strconv.Atoi(envValue)
		if err != nil {
			initializationErrors = append(initializationErrors,
				errors.NewInputError("value for environmental variable %s=%s can't be parsed into an integer", envVar, envValue),
			)
			return defaultValue
		}
		return envInt
	}
	addSuiteIDFlag(partitionCmd, &suiteID)

	defaultPartitionIndex := getEnvIntWithDefault("CAPTAIN_PARTITION_INDEX", -1)
	partitionCmd.Flags().IntVar(&pArgs.nodes.Index, "index", defaultPartitionIndex,
		"the 0-indexed index of a particular partition (required). Also set with CAPTAIN_PARTITION_INDEX")

	defaultPartitionTotal := getEnvIntWithDefault("CAPTAIN_PARTITION_TOTAL", 0)
	partitionCmd.Flags().IntVar(&pArgs.nodes.Total, "total", defaultPartitionTotal,
		"the total number of partitions (required). Also set with CAPTAIN_PARTITION_TOTAL")

	// it's a smell that we're using cliArgs here but I believe it's a major refactor to stop doing that.
	addShaFlag(partitionCmd, &cliArgs.GenericProvider.Sha)

	partitionCmd.Flags().StringVar(&pArgs.delimiter, "delimiter", " ", "the delimiter used to separate partitioned files")

	if defaultPartitionIndex < 0 {
		if err := partitionCmd.MarkFlagRequired("index"); err != nil {
			initializationErrors = append(initializationErrors, err)
		}
	}

	if defaultPartitionTotal < 1 {
		if err := partitionCmd.MarkFlagRequired("total"); err != nil {
			initializationErrors = append(initializationErrors, err)
		}
	}
	rootCmd.AddCommand(partitionCmd)
}
