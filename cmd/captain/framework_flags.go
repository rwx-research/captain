package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// TODO figure out how to handle shared state
var (
	providedFrameworkLanguage string
	providedFrameworkKind     string
)

func addFrameworkFlags(command *cobra.Command) {
	formattedKnownFrameworks := make([]string, len(v1.KnownFrameworks))
	for i, framework := range v1.KnownFrameworks {
		formattedKnownFrameworks[i] = fmt.Sprintf("  --language %v --framework %v", framework.Language, framework.Kind)
	}

	command.Flags().StringVar(
		&providedFrameworkLanguage,
		"language",
		"",
		fmt.Sprintf(
			"The programming language of the test suite (required if framework is set).\n"+
				"These can be set to anything, but Captain has specific handling for these combinations:\n%v",
			strings.Join(formattedKnownFrameworks, "\n"),
		),
	)
	command.Flags().StringVar(
		&providedFrameworkKind,
		"framework",
		"",
		fmt.Sprintf(
			"The framework of the test suite (required if language is set).\n"+
				"These can be set to anything, but Captain has specific handling for these combinations:\n%v",
			strings.Join(formattedKnownFrameworks, "\n"),
		),
	)
}

func bindFrameworkFlags(cfg Config) Config {
	if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
		if providedFrameworkKind != "" {
			suiteConfig.Results.Framework = providedFrameworkKind
		}

		if providedFrameworkLanguage != "" {
			suiteConfig.Results.Language = providedFrameworkLanguage
		}

		cfg.TestSuites[suiteID] = suiteConfig
	}

	return cfg
}
