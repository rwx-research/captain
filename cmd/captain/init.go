package main

import (
	"github.com/spf13/cobra"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/backend/local"
	"github.com/rwx-research/captain-cli/internal/backend/remote"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/logging"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/providers"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

var mutuallyExclusiveParsers []parsing.Parser = []parsing.Parser{
	new(parsing.DotNetxUnitParser),
	new(parsing.GoGinkgoParser),
	new(parsing.GoTestParser),
	new(parsing.JavaScriptCypressParser),
	new(parsing.JavaScriptJestParser),
	new(parsing.JavaScriptKarmaParser),
	new(parsing.JavaScriptMochaParser),
	new(parsing.JavaScriptPlaywrightParser),
	new(parsing.PythonPytestParser),
	new(parsing.RubyCucumberParser),
	new(parsing.RubyRSpecParser),
}

var frameworkParsers map[v1.Framework][]parsing.Parser = map[v1.Framework][]parsing.Parser{
	v1.DotNetxUnitFramework:          {new(parsing.DotNetxUnitParser)},
	v1.ElixirExUnitFramework:         {new(parsing.ElixirExUnitParser)},
	v1.GoGinkgoFramework:             {new(parsing.GoGinkgoParser)},
	v1.GoTestFramework:               {new(parsing.GoTestParser)},
	v1.JavaScriptCypressFramework:    {new(parsing.JavaScriptCypressParser)},
	v1.JavaScriptJestFramework:       {new(parsing.JavaScriptJestParser)},
	v1.JavaScriptKarmaFramework:      {new(parsing.JavaScriptKarmaParser)},
	v1.JavaScriptMochaFramework:      {new(parsing.JavaScriptMochaParser)},
	v1.JavaScriptPlaywrightFramework: {new(parsing.JavaScriptPlaywrightParser)},
	v1.PHPUnitFramework:              {new(parsing.PHPUnitParser)},
	v1.PythonPytestFramework:         {new(parsing.PythonPytestParser)},
	v1.PythonUnitTestFramework:       {new(parsing.PythonUnitTestParser)},
	v1.RubyCucumberFramework:         {new(parsing.RubyCucumberParser)},
	v1.RubyMinitestFramework:         {new(parsing.RubyMinitestParser)},
	v1.RubyRSpecFramework:            {new(parsing.RubyRSpecParser)},
}

var genericParsers []parsing.Parser = []parsing.Parser{
	new(parsing.RWXParser),
	new(parsing.JUnitParser),
}

// TODO: It looks like errors returned from this function are not getting logged correctly
// note: different commands have different requirements for the provider
// so they pass in their own validator accordingly
func initCLIService(providerValidator func(providers.Provider) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var apiClient backend.Client
		var err error

		cfg, err = InitConfig(cmd)
		if err != nil {
			return errors.WithStack(err)
		}

		logger := logging.NewProductionLogger()
		if cfg.Output.Debug {
			logger = logging.NewDebugLogger()
		}

		providerAdapter, err := cfg.ProvidersEnv.MakeProviderAdapter()
		if err != nil {
			return errors.Wrap(err, "failed to construct provider adapter")
		}
		err = providerValidator(providerAdapter)
		if err != nil {
			return err
		}

		if cfg.Secrets.APIToken != "" {
			apiClient, err = remote.NewClient(remote.ClientConfig{
				Debug:    cfg.Output.Debug,
				Host:     cfg.Cloud.APIHost,
				Insecure: cfg.Cloud.Insecure,
				Log:      logger,
				Token:    cfg.Secrets.APIToken,
				Provider: providerAdapter,
			})
		} else {
			var flakyTestsFilePath, testTimingsFilePath string
			flakyTestsFilePath, err = findInParentDir(flakyTestsFileName)
			if err != nil {
				flakyTestsFilePath = flakyTestsFileName
				logger.Warnf(
					"Unable to find existing flaky-tests.json file. Captain will create a new one at %q",
					flakyTestsFilePath,
				)
			}

			testTimingsFilePath, err = findInParentDir(testTimingsFileName)
			if err != nil {
				testTimingsFilePath = testTimingsFileName
				logger.Warnf(
					"Unable to find existing test-timings.json file. Captain will create a new one at %q",
					testTimingsFilePath,
				)
			}

			apiClient, err = local.NewClient(fs.Local{}, flakyTestsFilePath, testTimingsFilePath)
		}
		if err != nil {
			return errors.Wrap(err, "unable to create API client")
		}

		var frameworkKind, frameworkLanguage string
		if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
			frameworkKind = suiteConfig.Results.Framework
			frameworkLanguage = suiteConfig.Results.Language
		}

		parseConfig := parsing.Config{
			ProvidedFrameworkKind:     frameworkKind,
			ProvidedFrameworkLanguage: frameworkLanguage,
			MutuallyExclusiveParsers:  mutuallyExclusiveParsers,
			FrameworkParsers:          frameworkParsers,
			GenericParsers:            genericParsers,
			Logger:                    logger,
		}

		if err := parseConfig.Validate(); err != nil {
			return errors.Wrap(err, "invalid parser config")
		}

		captain = cli.Service{
			API:         apiClient,
			Log:         logger,
			FileSystem:  fs.Local{},
			TaskRunner:  exec.Local{},
			ParseConfig: parseConfig,
		}

		return nil
	}
}

// unsafeInitParsingOnly initializes an incomplete `captain` CLI service. This service is sufficient for running
// `captain parse`, but not for any other operation.
// It is considered unsafe since the captain CLI service might still expect a configured API at one point.
func unsafeInitParsingOnly(cmd *cobra.Command, args []string) error {
	var err error

	cfg, err = InitConfig(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	logger := logging.NewProductionLogger()
	if cfg.Output.Debug {
		logger = logging.NewDebugLogger()
	}

	var frameworkKind, frameworkLanguage string
	if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
		frameworkKind = suiteConfig.Results.Framework
		frameworkLanguage = suiteConfig.Results.Language
	}

	parseConfig := parsing.Config{
		ProvidedFrameworkKind:     frameworkKind,
		ProvidedFrameworkLanguage: frameworkLanguage,
		MutuallyExclusiveParsers:  mutuallyExclusiveParsers,
		FrameworkParsers:          frameworkParsers,
		GenericParsers:            genericParsers,
		Logger:                    logger,
	}
	if err := parseConfig.Validate(); err != nil {
		return errors.Wrap(err, "invalid parser config")
	}

	captain = cli.Service{
		Log:         logger,
		FileSystem:  fs.Local{},
		ParseConfig: parseConfig,
	}

	return nil
}
