package main

import (
	"path/filepath"
	"regexp"

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

var invalidSuiteIDRegexp = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// TODO: It looks like errors returned from this function are not getting logged correctly
// note: different commands have different requirements for the provider
// so they pass in their own validator accordingly
func initCLIService(
	cliArgs *CliArgs,
	providerValidator func(providers.Provider) error,
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var apiClient backend.Client

		cfg, err := InitConfig(cmd, cliArgs, args)
		if err != nil {
			return errors.WithDecoration(err)
		}

		if err = setConfig(cmd, cfg); err != nil {
			return errors.WithDecoration(err)
		}

		logger := logging.NewProductionLogger()
		if cfg.Output.Debug {
			logger = logging.NewDebugLogger()
		}

		providerAdapter, err := cfg.ProvidersEnv.MakeProviderAdapter()
		if err != nil {
			return errors.WithDecoration(errors.Wrap(err, "failed to construct provider adapter"))
		}
		err = providerValidator(providerAdapter)
		if err != nil {
			return errors.WithDecoration(err)
		}

		suiteID := cliArgs.RootCliArgs.suiteID

		if cfg.Secrets.APIToken != "" && !cfg.Cloud.Disabled {
			apiClient, err = remote.NewClient(remote.ClientConfig{
				Debug:    cfg.Output.Debug,
				Host:     cfg.Cloud.APIHost,
				Insecure: cfg.Cloud.Insecure,
				Log:      logger,
				Token:    cfg.Secrets.APIToken,
				Provider: providerAdapter,
			})
		} else {
			var flakesFilePath, quarantinesFilePath, timingsFilePath string
			if suiteID == "" {
				return errors.NewConfigurationError("Invalid suite-id", "The suite ID is empty.", "")
			}

			if !cfg.Cloud.Disabled {
				logger.Warnf("Unable to find RWX_ACCESS_TOKEN in the environment. Captain will default to OSS mode.")
				logger.Warnf("You can silence this warning by setting the following in the config file:")
				logger.Warnf("")
				logger.Warnf("cloud:")
				logger.Warnf("  disabled: true")
				logger.Warnf("")
			}
			if cfg.Secrets.APIToken != "" {
				logger.Warnf("Captain detected an RWX_ACCESS_TOKEN in your environment, however Cloud mode was disabled.")
				logger.Warnf("To start using Captain Cloud, please remove the 'cloud.disabled' setting in the config file.")
			}

			if invalidSuiteIDRegexp.Match([]byte(suiteID)) {
				return errors.NewConfigurationError(
					"Invalid suite-id",
					"A suite ID can only contain alphanumeric characters, `_` and `-`.",
					"Please make sure that the ID doesn't contain any special characters.",
				)
			}

			flakesFilePath, err = findInParentDir(filepath.Join(captainDirectory, suiteID, flakesFileName))
			if err != nil {
				flakesFilePath = filepath.Join(captainDirectory, suiteID, flakesFileName)
				logger.Warnf(
					"Unable to find existing flakes.yaml file for suite %q. Captain will create a new one at %q",
					suiteID, flakesFilePath,
				)
			}

			quarantinesFilePath, err = findInParentDir(filepath.Join(captainDirectory, suiteID, quarantinesFileName))
			if err != nil {
				quarantinesFilePath = filepath.Join(captainDirectory, suiteID, quarantinesFileName)
				logger.Warnf(
					"Unable to find existing quarantines.yaml for suite %q file. Captain will create a new one at %q",
					suiteID, quarantinesFilePath,
				)
			}

			timingsFilePath, err = findInParentDir(filepath.Join(captainDirectory, suiteID, timingsFileName))
			if err != nil {
				timingsFilePath = filepath.Join(captainDirectory, suiteID, timingsFileName)
				logger.Warnf(
					"Unable to find existing timings.yaml file. Captain will create a new one at %q",
					timingsFilePath,
				)
			}

			apiClient, err = local.NewClient(fs.Local{}, flakesFilePath, quarantinesFilePath, timingsFilePath)
		}
		if err != nil {
			return errors.WithDecoration(errors.Wrap(err, "unable to create API client"))
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
			return errors.WithDecoration(errors.Wrap(err, "invalid parser config"))
		}

		captain := cli.Service{
			API:         apiClient,
			Log:         logger,
			FileSystem:  fs.Local{},
			TaskRunner:  exec.Local{},
			ParseConfig: parseConfig,
		}

		if err := cli.SetService(cmd, captain); err != nil {
			return errors.WithStack(err)
		}

		return nil
	}
}

// unsafeInitParsingOnly initializes an incomplete `captain` CLI service. This service is sufficient for running
// `captain parse`, but not for any other operation.
// It is considered unsafe since the captain CLI service might still expect a configured API at one point.

func unsafeInitParsingOnly(cliArgs *CliArgs) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := InitConfig(cmd, cliArgs, args)
		if err != nil {
			return errors.WithStack(err)
		}

		if err := setConfig(cmd, cfg); err != nil {
			return errors.WithStack(err)
		}

		logger := logging.NewProductionLogger()
		if cfg.Output.Debug {
			logger = logging.NewDebugLogger()
		}

		var frameworkKind, frameworkLanguage string
		if suiteConfig, ok := cfg.TestSuites[cliArgs.RootCliArgs.suiteID]; ok {
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

		captain := cli.Service{
			Log:         logger,
			FileSystem:  fs.Local{},
			ParseConfig: parseConfig,
		}
		if err := cli.SetService(cmd, captain); err != nil {
			return errors.WithStack(err)
		}

		return nil
	}
}
