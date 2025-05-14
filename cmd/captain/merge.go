package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/reporting"
)

func createMergeCommand(cliArgs *CliArgs) *cobra.Command {
	cmd := cobra.Command{
		Use:   "merge results-glob [results-globs] <args>",
		Short: "Merge test results files",
		Long: "'captain merge' takes test results files produced by partitioned " +
			"executions and merges them into a single set of results.",
		Example: `  captain merge tmp/results/*.json`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: unsafeInitParsingOnly(cliArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := func() error {
				captain, err := cli.GetService(cmd)
				if err != nil {
					return errors.WithStack(err)
				}

				reporterFuncs := make(map[string]cli.Reporter)
				for _, r := range cliArgs.reporters {
					name, path, _ := strings.Cut(r, "=")

					switch name {
					case "rwx-v1-json":
						reporterFuncs[path] = reporting.WriteJSONSummary
					case "junit-xml":
						reporterFuncs[path] = reporting.WriteJUnitSummary
					case "markdown-summary":
						reporterFuncs[path] = reporting.WriteMarkdownSummary
					case "github-step-summary":
						stepSummaryPath := os.Getenv("GITHUB_STEP_SUMMARY")
						if stepSummaryPath == "" {
							captain.Log.Debug(
								"Skipping configuration of the 'github-step-summary' reporter " +
									"(the 'GITHUB_STEP_SUMMARY' environment variable is not set).",
							)
							continue
						}

						reporterFuncs[stepSummaryPath] = reporting.WriteMarkdownSummary
					default:
						return errors.NewConfigurationError(
							fmt.Sprintf("Unknown reporter %q", name),
							"Available reporters are 'rwx-v1-json', 'junit-xml', 'markdown-summary', and 'github-step-summary'.",
							"",
						)
					}
				}

				mergeConfig := cli.MergeConfig{
					ResultsGlobs: cliArgs.RootCliArgs.positionalArgs,
					PrintSummary: cliArgs.printSummary,
					Reporters:    reporterFuncs,
				}

				err = captain.Merge(cmd.Context(), mergeConfig)
				if _, ok := errors.AsConfigurationError(err); !ok {
					cmd.SilenceUsage = true
				}

				return errors.WithStack(err)
			}()
			if err != nil {
				return errors.WithDecoration(err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(
		&cliArgs.printSummary,
		"print-summary",
		false,
		"prints a summary of all tests to the console",
	)

	cmd.Flags().StringArrayVar(
		&cliArgs.reporters,
		"reporter",
		[]string{},
		"one or more `type=output_path` pairs to enable different reporting options.\n"+
			"Available reporters are 'rwx-v1-json', 'junit-xml', 'markdown-summary', and 'github-step-summary'.",
	)

	addFrameworkFlags(&cmd, &cliArgs.frameworkParams)

	return &cmd
}
