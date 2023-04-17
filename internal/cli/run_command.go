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
	commandArgs      []string
	shortCircuit     bool
	shortCircuitInfo string
}

func commandArgs(command string, args []string) ([]string, error) {
	commandArgs := make([]string, 0)

	if command != "" {
		parsedCommand, err := shellwords.Parse(command)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to parse %q into shell arguments", command)
		}
		commandArgs = append(commandArgs, parsedCommand...)
	}

	commandArgs = append(commandArgs, args...)

	if len(commandArgs) == 0 {
		return nil, errors.NewInputError("No command was provided")
	}

	return commandArgs, nil
}

func (s Service) makeRunCommand(ctx context.Context, cfg RunConfig) (RunCommand, error) {
	if !cfg.IsRunningPartition() {
		commandArgs, err := commandArgs(cfg.Command, cfg.Args)
		if err != nil {
			return RunCommand{}, err
		}
		return RunCommand{commandArgs: commandArgs, shortCircuit: false}, nil
	}

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

	commandArgs, err := commandArgs(partitionCommand, nil)
	if err != nil {
		return RunCommand{}, err
	}

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
		return RunCommand{commandArgs: commandArgs, shortCircuit: true, shortCircuitInfo: infoMessage}, nil
	}

	return RunCommand{commandArgs: commandArgs, shortCircuit: false}, nil
}
