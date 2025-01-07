package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

type frameworkParams struct {
	kind     string
	language string
}

func addFrameworkFlags(command *cobra.Command, frameworkParams *frameworkParams) {
	formattedKnownFrameworks := make([]string, len(v1.KnownFrameworks))
	for i, framework := range v1.KnownFrameworks {
		formattedKnownFrameworks[i] = fmt.Sprintf("  --language %v --framework %v", framework.Language, framework.Kind)
	}

	command.Flags().StringVar(
		&frameworkParams.language,
		"language",
		"",
		fmt.Sprintf(
			"The programming language of the test suite (required if framework is set).\n"+
				"These can be set to anything, but Captain has specific handling for these combinations:\n%v",
			strings.Join(formattedKnownFrameworks, "\n"),
		),
	)
	command.Flags().StringVar(
		&frameworkParams.kind,
		"framework",
		"",
		fmt.Sprintf(
			"The framework of the test suite (required if language is set).\n"+
				"These can be set to anything, but Captain has specific handling for these combinations:\n%v",
			strings.Join(formattedKnownFrameworks, "\n"),
		),
	)
}

func bindFrameworkFlags(cfg Config, frameworkParams frameworkParams, suiteID string) Config {
	if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
		if frameworkParams.kind != "" {
			suiteConfig.Results.Framework = frameworkParams.kind
		}

		if frameworkParams.language != "" {
			suiteConfig.Results.Language = frameworkParams.language
		}

		cfg.TestSuites[suiteID] = suiteConfig
	}

	return cfg
}
