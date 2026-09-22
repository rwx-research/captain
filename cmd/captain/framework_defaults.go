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
		suite.Command = `dotnet test --logger "xunit;LogFileName=xunit.xml" --results-directory "."`
		suite.Results.Path = "xunit.xml"
		suite.Retries.Command = suite.Command + ` --filter '{{ filter }}'`
	case v1.ElixirExUnitFramework:
		suite.Command = "mix test"
		suite.Results.Path = "tmp/junit/*.xml"
		suite.Retries.Command = "mix test {{ tests }}"
		suite.Partition.Command = "mix test {{ testFiles }}"
		suite.Partition.Globs = []string{"test/**/*_test.exs"}
	case v1.GoGinkgoFramework:
		suite.Command = "ginkgo run --keep-going --json-report ./ginkgo.json ./..."
		suite.Results.Path = "ginkgo.json"
		suite.Retries.Command = "ginkgo run {{ tests }} --keep-going --json-report ./ginkgo.json ./..."
	case v1.GoTestFramework:
		suite.Command = "gotestsum --raw-command --jsonfile go-test.json -- go test ./... -json -count=1"
		suite.Results.Path = "go-test.json"
		suite.Retries.Command = "gotestsum --raw-command --jsonfile go-test.json -- " +
			"go test {{ package }} -run '{{ run }}' -json -count=1"
	case v1.JavaScriptBunFramework:
		suite.Command = "bun test --reporter=junit --reporter-outfile bun.xml"
		suite.Results.Path = "bun.xml"
		suite.Retries.Command = "bun test '{{ file }}' --test-name-pattern '{{ testNamePattern }}' " +
			"--reporter=junit --reporter-outfile bun.xml"
		suite.Partition.Command = "bun test {{ testFiles }} --reporter=junit --reporter-outfile bun.xml"
		suite.Partition.Globs = []string{"**/*.test.ts"}
	case v1.JavaScriptCucumberFramework:
		suite.Command = "npx cucumber-js --format json:cucumber.json --format summary"
		suite.Results.Path = "cucumber.json"
		suite.Retries.Command = suite.Command + " {{ scenarios }}"
		suite.Partition.Command = suite.Command + " {{ testFiles }}"
		suite.Partition.Globs = []string{"features/**/*.feature"}
	case v1.JavaScriptCypressFramework:
		suite.Command = "npx cypress run"
		suite.Results.Path = "tmp/rwx/*.json"
		suite.Retries.Command = "npx cypress run --spec '{{ spec }}' --env {{ grep }}"
		suite.Partition.Command = "npx cypress run --spec {{ testFiles }}"
		suite.Partition.Delimiter = ","
		suite.Partition.Globs = []string{"cypress/**/*.cy.js"}
	case v1.JavaScriptJestFramework:
		suite.Command = "npx jest --json --testLocationInResults --outputFile jest.json"
		suite.Results.Path = "jest.json"
		suite.Retries.Command = "npx jest --testPathPattern '{{ testPathPattern }}' " +
			"--testNamePattern '{{ testNamePattern }}' --json --testLocationInResults --outputFile jest.json"
		suite.Partition.Command = "npx jest --testPathPattern {{ testFiles }} " +
			"--json --testLocationInResults --outputFile jest.json"
		suite.Partition.Delimiter = "|"
		suite.Partition.Globs = []string{"**/*.test.js"}
	case v1.JavaScriptKarmaFramework:
		suite.Command = "npx karma start --single-run"
		suite.Results.Path = "tmp/karma.json"
	case v1.JavaScriptMochaFramework:
		const reporter = " --reporter @rwx-research/mocha-multi-reporters --reporter-options configFile=multi-reporters.json"
		suite.Command = "npx mocha 'test/**/*.js'" + reporter
		suite.Results.Path = "tmp/mocha.json"
		suite.Retries.Command = "npx mocha '{{ file }}' --grep '{{ grep }}'" + reporter
		suite.Partition.Command = "npx mocha {{ testFiles }}" + reporter
		suite.Partition.Globs = []string{"test/**/*.js"}
	case v1.JavaScriptPlaywrightFramework:
		suite.Command = "bash -c 'PLAYWRIGHT_JSON_OUTPUT_NAME=playwright.json npx playwright test --reporter=json'"
		suite.Results.Path = "playwright.json"
		suite.Retries.Command = `bash -c "PLAYWRIGHT_JSON_OUTPUT_NAME=playwright.json npx playwright test ` +
			`{{ tests }} --project '{{ project }}' --reporter=json"`
		suite.Partition.Command = "bash -c 'PLAYWRIGHT_JSON_OUTPUT_NAME=playwright.json " +
			"npx playwright test {{ testFiles }} --reporter=json'"
		suite.Partition.Globs = []string{"tests/**/*.spec.js"}
	case v1.JavaScriptTestCafeFramework:
		suite.Command = "npx testcafe chrome:headless tests/ --reporter spec,json:testcafe.json"
		suite.Results.Path = "testcafe.json"
		suite.Retries.Command = "npx testcafe chrome:headless '{{ file }}' -T '{{ grep }}' --reporter spec,json:testcafe.json"
		suite.Partition.Command = "npx testcafe chrome:headless {{ testFiles }} --reporter spec,json:testcafe.json"
		suite.Partition.Globs = []string{"tests/**/*.testcafe.ts"}
	case v1.JavaScriptVitestFramework:
		suite.Command = "npx vitest run --reporter=default --reporter=json --outputFile=./vitest.json"
		suite.Results.Path = "vitest.json"
		suite.Retries.Command = "npx vitest run '{{ file }}' --testNamePattern '{{ testNamePattern }}' " +
			"--reporter=default --reporter=json --outputFile=./vitest.json"
		suite.Partition.Command = "npx vitest run {{ testFiles }} " +
			"--reporter=default --reporter=json --outputFile=./vitest.json"
		suite.Partition.Globs = []string{"**/*.test.js"}
	case v1.PHPUnitFramework:
		suite.Command = "vendor/bin/phpunit --log-junit phpunit.xml tests/"
		suite.Results.Path = "phpunit.xml"
		suite.Retries.Command = "vendor/bin/phpunit --log-junit phpunit.xml --filter '{{ filter }}' '{{ file }}'"
	case v1.PythonPytestFramework:
		suite.Command = "pytest --report-log=log.json"
		suite.Results.Path = "log.json"
		suite.Retries.Command = suite.Command + " {{ tests }}"
		suite.Partition.Command = suite.Command + " {{ testFiles }}"
		suite.Partition.Globs = []string{"test/**/test_*.py"}
	case v1.PythonUnitTestFramework:
		suite.Command = "python -m xmlrunner -o ."
		suite.Results.Path = "TEST-*.xml"
		suite.Retries.Command = suite.Command + " {{ tests }}"
		suite.Partition.Command = suite.Command + " {{ testFiles }}"
		suite.Partition.Globs = []string{"test/**/test_*.py"}
	case v1.RubyCucumberFramework:
		suite.Command = "bundle exec cucumber --format json --out cucumber.json --format pretty"
		suite.Results.Path = "cucumber.json"
		suite.Retries.Command = suite.Command + " {{ scenarios }}"
		suite.Partition.Command = suite.Command + " {{ testFiles }}"
		suite.Partition.Globs = []string{"features/**/*.feature"}
	case v1.RubyMinitestFramework:
		suite.Command = "bin/rails test"
		suite.Results.Path = "tmp/reports/*.xml"
		suite.Retries.Command = "bin/rails test {{ tests }}"
		suite.Partition.Command = "bin/rails test {{ testFiles }}"
		suite.Partition.Globs = []string{"test/**/*_test.rb"}
	case v1.RubyRSpecFramework:
		suite.Command = "bundle exec rspec --format json --out rspec.json --format progress"
		suite.Results.Path = "rspec.json"
		suite.Retries.Command = suite.Command + " {{ tests }}"
		suite.Partition.Command = suite.Command + " {{ testFiles }}"
		suite.Partition.Globs = []string{"spec/**/*_spec.rb"}
	default:
		return suite, nil
	}

	suite.Results.Language = string(framework.Language)
	suite.Results.Framework = string(framework.Kind)
	suite.Output.PrintSummary = true
	return suite, nil
}
