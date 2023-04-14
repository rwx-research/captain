package cli

import (
	"context"
	"fmt"

	"github.com/mattn/go-shellwords"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/runpartition"
	"github.com/rwx-research/captain-cli/internal/templating"
)

// RunCommand represents the command that captain run ultimately execute.
// Typically this is executing the underlying test framework.
type RunCommand struct {
	command          string
	args             []string
	shortCircuit     bool
	shortCircuitInfo string
}

func (c RunCommand) commandArgs() ([]string, error) {
	commandArgs := make([]string, 0)

	if c.command != "" {
		parsedCommand, err := shellwords.Parse(c.command)
		if err != nil {
			return commandArgs, errors.Wrapf(err, "Unable to parse %q into shell arguments", c.command)
		}
		commandArgs = append(commandArgs, parsedCommand...)
	}

	if len(c.args) > 0 {
		commandArgs = append(commandArgs, c.args...)
	}

	if len(commandArgs) == 0 {
		return commandArgs, errors.NewInputError("No command was provided")
	}

	return commandArgs, nil
}

func (s Service) makeRunCommand(ctx context.Context, cfg RunConfig) (RunCommand, error) {
	if cfg.IsRunningPartition() {
		partitionResult, err := s.calculatePartition(ctx, cfg.PartitionConfig)
		if err != nil {
			return RunCommand{}, errors.WithStack(err)
		}
		partitionedTestFilePaths := partitionResult.partition.TestFilePaths

		// compile template
		compiledTemplate, err := templating.CompileTemplate(cfg.PartitionCommandTemplate)
		if err != nil {
			return RunCommand{}, errors.WithStack(err)
		}

		// validate template
		substitution := runpartition.DelimiterSubstitution{Delimiter: cfg.PartitionConfig.Delimiter}
		if err := substitution.ValidateTemplate(compiledTemplate); err != nil {
			return RunCommand{}, errors.WithStack(err)
		}

		// substitute template keywords with values
		substitutionValueLookup, err := substitution.SubstitutionLookupFor(compiledTemplate, partitionedTestFilePaths)
		if err != nil {
			return RunCommand{}, errors.WithStack(err)
		}
		partitionCommand := compiledTemplate.Substitute(substitutionValueLookup)

		if len(partitionedTestFilePaths) == 0 {
			infoMessage := fmt.Sprintf(
				"Partition %v contained no test files. %d/%d partitions were utilized. "+
					"We recommend you set --partition-total no more than %d",
				cfg.PartitionConfig.PartitionNodes,
				partitionResult.utilizedPartitionCount,
				cfg.PartitionConfig.PartitionNodes.Total,
				partitionResult.utilizedPartitionCount,
			)
			// short circuit to avoid running the entire test suite in a single partition (e.g empty partition)
			return RunCommand{command: partitionCommand, shortCircuit: true, shortCircuitInfo: infoMessage}, nil
		}

		return RunCommand{command: partitionCommand, shortCircuit: false}, nil
	}

	return RunCommand{command: cfg.Command, args: cfg.Args, shortCircuit: false}, nil
}
