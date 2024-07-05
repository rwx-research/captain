package main

import (
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

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
	parsing.DotNetxUnitParser{},
	parsing.GoGinkgoParser{},
	parsing.GoTestParser{},
	parsing.JavaScriptCypressParser{},
	parsing.JavaScriptJestParser{},
	parsing.JavaScriptKarmaParser{},
	parsing.JavaScriptMochaParser{},
	parsing.JavaScriptPlaywrightParser{},
	parsing.PythonPytestParser{},
	parsing.RubyRSpecParser{},
}

var frameworkParsers map[v1.Framework][]parsing.Parser = map[v1.Framework][]parsing.Parser{
	v1.DotNetxUnitFramework:          {parsing.DotNetxUnitParser{}},
	v1.ElixirExUnitFramework:         {parsing.ElixirExUnitParser{}},
	v1.GoGinkgoFramework:             {parsing.GoGinkgoParser{}},
	v1.GoTestFramework:               {parsing.GoTestParser{}},
	v1.JavaScriptCucumberFramework:   {parsing.JavaScriptCucumberJSONParser{}},
	v1.JavaScriptCypressFramework:    {parsing.JavaScriptCypressParser{}},
	v1.JavaScriptJestFramework:       {parsing.JavaScriptJestParser{}},
	v1.JavaScriptKarmaFramework:      {parsing.JavaScriptKarmaParser{}},
	v1.JavaScriptMochaFramework:      {parsing.JavaScriptMochaParser{}},
	v1.JavaScriptPlaywrightFramework: {parsing.JavaScriptPlaywrightParser{}},
	v1.PHPUnitFramework:              {parsing.PHPUnitParser{}},
	v1.PythonPytestFramework:         {parsing.PythonPytestParser{}},
	v1.PythonUnitTestFramework:       {parsing.PythonUnitTestParser{}},
	v1.RubyCucumberFramework:         {parsing.RubyCucumberParser{}},
	v1.RubyMinitestFramework:         {parsing.RubyMinitestParser{}},
	v1.RubyRSpecFramework:            {parsing.RubyRSpecParser{}},
}

var genericParsers []parsing.Parser = []parsing.Parser{
	parsing.RWXParser{},
	parsing.JUnitTestsuitesParser{},
	parsing.JUnitTestsuiteParser{},
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
		if err := extractSuiteIDFromPositionalArgs(&cliArgs.RootCliArgs, args); err != nil {
			return err
		}

		cfg, err := InitConfig(cmd, *cliArgs)
		if err != nil {
			return errors.WithDecoration(err)
		}

		return initCliServiceWithConfig(cmd, cfg, cliArgs.RootCliArgs.suiteID, providerValidator)
	}
}

func initCliServiceWithConfig(
	cmd *cobra.Command, cfg Config, suiteID string, providerValidator func(providers.Provider) error,
) error {
	err := func() error {
		if suiteID == "" {
			return errors.NewConfigurationError("Invalid suite-id", "The suite ID is empty.", "")
		}

		if invalidSuiteIDRegexp.Match([]byte(suiteID)) {
			return errors.NewConfigurationError(
				"Invalid suite-id",
				"A suite ID can only contain alphanumeric characters, `_` and `-`.",
				"Please make sure that the ID doesn't contain any special characters.",
			)
		}

		logger := logging.NewProductionLogger()
		if cfg.Output.Debug {
			logger = logging.NewDebugLogger()
		}

		apiClient, err := makeAPIClient(cfg, providerValidator, logger, suiteID)
		if err != nil {
			return errors.Wrap(err, "unable to create API client")
		}

		var parseConfig parsing.Config
		if suiteConfig, ok := cfg.TestSuites[suiteID]; ok {
			parseConfig = parsing.Config{
				ProvidedFrameworkKind:     suiteConfig.Results.Framework,
				ProvidedFrameworkLanguage: suiteConfig.Results.Language,
				MutuallyExclusiveParsers:  mutuallyExclusiveParsers,
				FrameworkParsers:          frameworkParsers,
				GenericParsers:            genericParsers,
				Logger:                    logger,
			}
		}

		if err := parseConfig.Validate(); err != nil {
			return errors.Wrap(err, "invalid parser config")
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
	}()
	if err != nil {
		return errors.WithDecoration(err)
	}
	return nil
}

// unsafeInitParsingOnly initializes an incomplete `captain` CLI service. This service is sufficient for running
// `captain parse`, but not for any other operation.
// It is considered unsafe since the captain CLI service might still expect a configured API at one point.

func unsafeInitParsingOnly(cliArgs *CliArgs) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := func() error {
			cliArgs.RootCliArgs.positionalArgs = args
			if cliArgs.RootCliArgs.suiteID == "" {
				cliArgs.RootCliArgs.suiteID = "placeholder"
			}

			cfg, err := InitConfig(cmd, *cliArgs)
			if err != nil {
				return errors.WithStack(err)
			}

			logger := logging.NewProductionLogger()
			if cfg.Output.Debug {
				logger = logging.NewDebugLogger()
			}

			var parseConfig parsing.Config
			if suiteConfig, ok := cfg.TestSuites[cliArgs.RootCliArgs.suiteID]; ok {
				parseConfig = parsing.Config{
					ProvidedFrameworkKind:     suiteConfig.Results.Framework,
					ProvidedFrameworkLanguage: suiteConfig.Results.Language,
					MutuallyExclusiveParsers:  mutuallyExclusiveParsers,
					FrameworkParsers:          frameworkParsers,
					GenericParsers:            genericParsers,
					Logger:                    logger,
				}
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
		}()
		if err != nil {
			return errors.WithDecoration(err)
		}
		return nil
	}
}

func makeAPIClient(
	cfg Config, providerValidator func(providers.Provider) error, logger *zap.SugaredLogger, suiteID string,
) (backend.Client, error) {
	wrapError := func(a backend.Client, b error) (backend.Client, error) {
		return a, errors.WithStack(b)
	}
	if cfg.Secrets.APIToken != "" && !cfg.Cloud.Disabled {
		provider, err := cfg.ProvidersEnv.MakeProvider()
		if err != nil {
			return nil, errors.Wrap(err, "failed to construct provider")
		}
		err = providerValidator(provider)
		if err != nil {
			return nil, err
		}

		return wrapError(remote.NewClient(remote.ClientConfig{
			Debug:    cfg.Output.Debug,
			Host:     cfg.Cloud.APIHost,
			Insecure: cfg.Cloud.Insecure,
			Log:      logger,
			Token:    cfg.Secrets.APIToken,
			Provider: provider,
		}))
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

	flakesFilePath, err := findInParentDir(filepath.Join(captainDirectory, suiteID, flakesFileName))
	if err != nil {
		flakesFilePath = filepath.Join(captainDirectory, suiteID, flakesFileName)
		logger.Warnf(
			"Unable to find existing flakes.yaml file for suite %q. Captain will create a new one at %q",
			suiteID, flakesFilePath,
		)
	}

	quarantinesFilePath, err := findInParentDir(filepath.Join(captainDirectory, suiteID, quarantinesFileName))
	if err != nil {
		quarantinesFilePath = filepath.Join(captainDirectory, suiteID, quarantinesFileName)
		logger.Warnf(
			"Unable to find existing quarantines.yaml for suite %q file. Captain will create a new one at %q",
			suiteID, quarantinesFilePath,
		)
	}

	timingsFilePath, err := findInParentDir(filepath.Join(captainDirectory, suiteID, timingsFileName))
	if err != nil {
		timingsFilePath = filepath.Join(captainDirectory, suiteID, timingsFileName)
		logger.Warnf(
			"Unable to find existing timings.yaml file. Captain will create a new one at %q",
			timingsFilePath,
		)
	}

	return wrapError(local.NewClient(fs.Local{}, flakesFilePath, quarantinesFilePath, timingsFilePath))
}
