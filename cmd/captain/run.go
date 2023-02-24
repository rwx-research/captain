package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/reporting"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

var (
	testResults              string
	failOnUploadError        bool
	reporters                []string
	retries                  int
	retryCommandTemplate     string
	substitutionsByFramework = map[v1.Framework]targetedretries.Substitution{
		v1.DotNetxUnitFramework:          new(targetedretries.DotNetxUnitSubstitution),
		v1.ElixirExUnitFramework:         new(targetedretries.ElixirExUnitSubstitution),
		v1.GoGinkgoFramework:             new(targetedretries.GoGinkgoSubstitution),
		v1.GoTestFramework:               new(targetedretries.GoTestSubstitution),
		v1.JavaScriptCypressFramework:    new(targetedretries.JavaScriptCypressSubstitution),
		v1.JavaScriptJestFramework:       new(targetedretries.JavaScriptJestSubstitution),
		v1.JavaScriptMochaFramework:      new(targetedretries.JavaScriptMochaSubstitution),
		v1.JavaScriptPlaywrightFramework: new(targetedretries.JavaScriptPlaywrightSubstitution),
		v1.PHPUnitFramework:              new(targetedretries.PHPUnitSubstitution),
		v1.PythonPytestFramework:         new(targetedretries.PythonPytestSubstitution),
		v1.PythonUnitTestFramework:       new(targetedretries.PythonUnitTestSubstitution),
		v1.RubyCucumberFramework:         new(targetedretries.RubyCucumberSubstitution),
		v1.RubyMinitestFramework:         new(targetedretries.RubyMinitestSubstitution),
		v1.RubyRSpecFramework:            new(targetedretries.RubyRSpecSubstitution),
	}
	printSummary bool

	runCmd = &cobra.Command{
		Use:     "run",
		Short:   "Execute a build- or test-suite",
		Long:    descriptionRun,
		PreRunE: initCLIService,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.WithStack(cmd.Usage())
			}

			reporterFuncs := make(map[string]cli.Reporter)
			for _, reporter := range reporters {
				name, path, found := strings.Cut(reporter, "=")
				if !found {
					return errors.NewConfigurationError("Invalid reporter syntax %q. Expected `type=output_path`", reporter)
				}

				switch name {
				case "rwx-v1-json":
					reporterFuncs[path] = reporting.WriteJSONSummary
				case "junit-xml":
					reporterFuncs[path] = reporting.WriteJUnitSummary
				default:
					return errors.NewConfigurationError("Unknown reporter %q.", name)
				}
			}

			runConfig := cli.RunConfig{
				Args:                     args,
				TestResultsFileGlob:      testResults,
				FailOnUploadError:        failOnUploadError,
				Reporters:                reporterFuncs,
				Retries:                  retries,
				RetryCommandTemplate:     retryCommandTemplate,
				SuiteID:                  suiteID,
				SubstitutionsByFramework: substitutionsByFramework,
				PrintSummary:             printSummary,
			}

			return errors.WithStack(captain.RunSuite(cmd.Context(), runConfig))
		},
	}
)

func init() {
	runCmd.Flags().StringVar(
		&testResults,
		"test-results",
		"",
		"a filepath to a test result - supports globs for multiple result files",
	)

	runCmd.Flags().BoolVar(
		&failOnUploadError,
		"fail-on-upload-error",
		false,
		"return a non-zero exit code in case the test results upload fails",
	)

	runCmd.Flags().StringArrayVar(
		&reporters,
		"reporter",
		[]string{},
		"one or more `type=output_path` pairs to enable different reporting options. "+
			"Available reporter types are `rwx-v1-json` and `junit-xml ",
	)

	runCmd.Flags().IntVar(
		&retries,
		"retries",
		0,
		"the number of times failed tests should be retried "+
			"(e.g. --retries 2 would mean a maximum of 3 attempts of any given test) (required if --retry-command is passed)",
	)

	formattedSubstitutionExamples := make([]string, len(substitutionsByFramework))
	i := 0
	for framework, substitution := range substitutionsByFramework {
		formattedSubstitutionExamples[i] = fmt.Sprintf("  %v: --retry-command \"%v\"", framework, substitution.Example())
		i++
	}
	sort.SliceStable(formattedSubstitutionExamples, func(i, j int) bool {
		return strings.ToLower(formattedSubstitutionExamples[i]) < strings.ToLower(formattedSubstitutionExamples[j])
	})

	runCmd.Flags().StringVar(
		&retryCommandTemplate,
		"retry-command",
		"",
		fmt.Sprintf(
			"the command that will be run to execute a subset of your tests while retrying "+
				"(required if --retries is passed)\n"+
				"Examples:\n%v",
			strings.Join(formattedSubstitutionExamples, "\n"),
		),
	)

	runCmd.Flags().BoolVar(
		&printSummary,
		"print-summary",
		false,
		"prints a summary of all tests to the console",
	)

	runCmd.MarkFlagsRequiredTogether("retries", "retry-command")

	addFrameworkFlags(runCmd)

	rootCmd.AddCommand(runCmd)
}
