package main

import (
	"fmt"
	"strings"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

func defaultRunSuite(params frameworkParams) (cli.SuiteConfig, error) {
	var suite cli.SuiteConfig
	if params.kind == "" {
		return suite, nil
	}

	if strings.TrimSpace(params.language) == "" {
		var languages []string
		for _, framework := range v1.KnownFrameworks {
			if strings.EqualFold(string(framework.Kind), strings.TrimSpace(params.kind)) {
				languages = append(languages, string(framework.Language))
			}
		}
		if len(languages) > 1 {
			return suite, errors.NewConfigurationError(
				"Ambiguous test framework",
				fmt.Sprintf("The %s framework supports multiple languages: %s.", params.kind, strings.Join(languages, ", ")),
				"Specify --language to select the framework defaults.",
			)
		}
		if len(languages) == 1 {
			params.language = languages[0]
		}
	}

	framework := v1.CoerceFramework(params.language, params.kind)
	// These commands follow https://www.rwx.com/docs/captain/test-frameworks.
	// Reporter-configured paths stay aligned with the documented reporter setup.
	switch framework {
	case v1.DotNetxUnitFramework:
		suite = defaultDotNetxUnitSuite()
	case v1.ElixirExUnitFramework:
		suite = defaultElixirExUnitSuite()
	case v1.GoGinkgoFramework:
		suite = defaultGoGinkgoSuite()
	case v1.GoTestFramework:
		suite = defaultGoTestSuite()
	case v1.JavaScriptBunFramework:
		suite = defaultJavaScriptBunSuite()
	case v1.JavaScriptCucumberFramework:
		suite = defaultJavaScriptCucumberSuite()
	case v1.JavaScriptCypressFramework:
		suite = defaultJavaScriptCypressSuite()
	case v1.JavaScriptJestFramework:
		suite = defaultJavaScriptJestSuite()
	case v1.JavaScriptKarmaFramework:
		suite = defaultJavaScriptKarmaSuite()
	case v1.JavaScriptMochaFramework:
		suite = defaultJavaScriptMochaSuite()
	case v1.JavaScriptPlaywrightFramework:
		suite = defaultJavaScriptPlaywrightSuite()
	case v1.JavaScriptTestCafeFramework:
		suite = defaultJavaScriptTestCafeSuite()
	case v1.JavaScriptVitestFramework:
		suite = defaultJavaScriptVitestSuite()
	case v1.PHPUnitFramework:
		suite = defaultPHPUnitSuite()
	case v1.PythonPytestFramework:
		suite = defaultPythonPytestSuite()
	case v1.PythonUnitTestFramework:
		suite = defaultPythonUnitTestSuite()
	case v1.RubyCucumberFramework:
		suite = defaultRubyCucumberSuite()
	case v1.RubyMinitestFramework:
		suite = defaultRubyMinitestSuite()
	case v1.RubyRSpecFramework:
		suite = defaultRubyRSpecSuite()
	default:
		return suite, nil
	}

	suite.Results.Language = string(framework.Language)
	suite.Results.Framework = string(framework.Kind)
	suite.Output.PrintSummary = true
	return suite, nil
}
