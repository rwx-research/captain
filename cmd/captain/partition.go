package main

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/config"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

type partitionArgs struct {
	nodes     config.PartitionNodes
	delimiter string
}

func configurePartitionCmd(rootCmd *cobra.Command, cliArgs *CliArgs) error {
	var pArgs partitionArgs
	// returns a negative value if the environmental variable is not set
	getEnvAsInt := func(name string) (int, error) {
		value := os.Getenv(name)
		if value == "" {
			return -1, nil
		}

		i, err := strconv.Atoi(value)
		if err != nil {
			return -1,
				errors.NewInputError("value for environmental variable %s=%s can't be parsed into an integer", name, value)
		}

		return i, nil
	}

	partitionCmd := &cobra.Command{
		Use: "partition [--help] [--config-file=<path>] [--delimiter=<delim>] [--sha=<sha>] --suite-id=<suite> --index=<i> " +
			"--total=<total> <args>",
		Short: "Partitions a test suite using historical file timings recorded by Captain",
		Long: "'captain partition' can be used to split up your test suite by test file, leveraging test file timings " +
			"recorded in captain.",
		Example: "" +
			"  bundle exec rspec $(captain partition your-project-rspec --index 0 --total 2 spec/**/*_spec.rb)\n" +
			"  bundle exec rspec $(captain partition your-project-rspec --index 1 --total 2 spec/**/*_spec.rb)",
		Args:                  cobra.MinimumNArgs(1),
		DisableFlagsInUseLine: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := func() error {
				if err := extractSuiteIDFromPositionalArgs(&cliArgs.RootCliArgs, args); err != nil {
					return err
				}

				cfg, err := InitConfig(cmd, *cliArgs)
				if err != nil {
					return err
				}

				provider, err := cfg.ProvidersEnv.MakeProvider()
				if err != nil {
					return errors.Wrap(err, "failed to construct provider")
				}

				if pArgs.nodes.Index < 0 {
					if provider.PartitionNodes.Index < 0 {
						return errors.NewConfigurationError(
							"Partition index invalid.",
							"Partition index must be 0 or greater.",
							"You can set the partition index by using the --index flag or the CAPTAIN_PARTITION_INDEX environment variable.",
						)
					}
					pArgs.nodes.Index = provider.PartitionNodes.Index
				}

				if pArgs.nodes.Total < 0 {
					if provider.PartitionNodes.Total < 1 {
						return errors.NewConfigurationError(
							"Partition total invalid.",
							"Partition total must be 1 or greater.",
							"You can set the partition index by using the --total flag or the CAPTAIN_PARTITION_TOTAL environment variable.",
						)
					}
					pArgs.nodes.Total = provider.PartitionNodes.Total
				}

				return initCliServiceWithConfig(cmd, cfg, cliArgs.RootCliArgs.suiteID, func(p providers.Provider) error {
					if p.CommitSha == "" {
						return errors.NewConfigurationError(
							"Missing commit SHA",
							"Captain requires a commit SHA in order to track test runs correctly.",
							"You can specify the SHA by using the --sha flag or the CAPTAIN_SHA environment variable",
						)
					}
					return nil
				})
			}()
			if err != nil {
				return errors.WithDecoration(err)
			}
			return nil
		},

		RunE: func(cmd *cobra.Command, _ []string) error {
			args := cliArgs.RootCliArgs.positionalArgs
			captain, err := cli.GetService(cmd)
			if err != nil {
				return errors.WithStack(err)
			}
			err = captain.Partition(cmd.Context(), cli.PartitionConfig{
				SuiteID:        cliArgs.RootCliArgs.suiteID,
				TestFilePaths:  args,
				PartitionNodes: pArgs.nodes,
				Delimiter:      pArgs.delimiter,
			})
			return errors.WithStack(err)
		},
	}

	defaultPartitionIndex, err := getEnvAsInt("CAPTAIN_PARTITION_INDEX")
	if err != nil {
		return err
	}

	partitionCmd.Flags().IntVar(
		&pArgs.nodes.Index, "index", defaultPartitionIndex, "the 0-indexed index of a particular partition",
	)

	defaultPartitionTotal, err := getEnvAsInt("CAPTAIN_PARTITION_TOTAL")
	if err != nil {
		return err
	}
	partitionCmd.Flags().IntVar(&pArgs.nodes.Total, "total", defaultPartitionTotal, "the total number of partitions")

	// it's a smell that we're using cliArgs here but I believe it's a major refactor to stop doing that.
	addShaFlag(partitionCmd, &cliArgs.GenericProvider.Sha)

	defaultDelimiter := os.Getenv("CAPTAIN_DELIMITER")
	if defaultDelimiter == "" {
		defaultDelimiter = " "
	}

	partitionCmd.Flags().StringVar(&pArgs.delimiter, "delimiter", defaultDelimiter,
		"the delimiter used to separate partitioned files.\n"+
			"It can also be set using the env var CAPTAIN_DELIMITER.")

	rootCmd.AddCommand(partitionCmd)
	return nil
}
