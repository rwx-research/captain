package main_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	captain "github.com/rwx-research/captain-cli/cmd/captain"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/runpartition"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI-only framework defaults", func() {
	configure := func(command, config string, flags ...string) (captain.Config, error) {
		configPath := filepath.Join(GinkgoT().TempDir(), "config.yaml")
		if config != "" {
			Expect(os.WriteFile(configPath, []byte(config), 0o600)).To(Succeed())
		}
		var args captain.CliArgs
		cmd := &cobra.Command{Use: command}
		cmd.SetContext(context.Background())
		Expect(captain.ConfigureRootCmd(cmd, &args)).To(Succeed())
		Expect(captain.AddFlags(cmd, &args)).To(Succeed())
		flags = append([]string{"--suite-id", "suite", "--config-file", configPath}, flags...)
		Expect(cmd.ParseFlags(flags)).To(Succeed())
		return captain.InitConfig(cmd, args)
	}

	It("uses the requested RSpec defaults without a config file", func() {
		cfg, err := configure("run", "", "--framework", "rspec")
		Expect(err).NotTo(HaveOccurred())
		suite := cfg.TestSuites["suite"]
		Expect(suite.Results.Language).To(Equal("Ruby"))
		Expect(suite.Results.Framework).To(Equal("rspec"))
		Expect(suite.Command).To(Equal("bundle exec rspec --format json --out rspec.json --format progress"))
		Expect(suite.Retries.Command).To(Equal(
			"bundle exec rspec --format json --out rspec.json --format progress {{ tests }}"))
		Expect(suite.Partition.Command).To(Equal(
			"bundle exec rspec --format json --out rspec.json --format progress {{ testFiles }}"))
		Expect(suite.Partition.Globs).To(Equal([]string{"spec/**/*_spec.rb"}))
		Expect(suite.Partition.Delimiter).To(Equal(" "))
		Expect(suite.Results.Path).To(Equal("rspec.json"))
		Expect(suite.Output.PrintSummary).To(BeTrue())
		Expect(suite.Retries.Attempts).To(Equal(-1))
	})

	DescribeTable("uses the documented framework commands and result paths",
		func(kind, language, command, results, glob, delimiter string) {
			flags := []string{"--framework", kind}
			if kind == "cucumber" {
				flags = append(flags, "--language", language)
			}
			cfg, err := configure("run", "", flags...)
			Expect(err).NotTo(HaveOccurred())
			suite := cfg.TestSuites["suite"]
			Expect(strings.ToLower(suite.Results.Language)).To(Equal(language))
			Expect(suite.Command).To(Equal(command))
			Expect(suite.Results.Path).To(Equal(results))
			Expect(suite.Output.PrintSummary).To(BeTrue())
			Expect(suite.Partition.Delimiter).To(Equal(delimiter))
			if glob == "" {
				Expect(suite.Partition.Globs).To(BeEmpty())
				Expect(suite.Partition.Command).To(BeEmpty())
			} else {
				Expect(suite.Partition.Globs).To(Equal([]string{glob}))
				template, err := templating.CompileTemplate(suite.Partition.Command)
				Expect(err).NotTo(HaveOccurred())
				Expect((runpartition.DelimiterSubstitution{Delimiter: delimiter}).ValidateTemplate(template)).To(Succeed())
			}
			if kind == "karma" {
				Expect(suite.Retries.Command).To(BeEmpty())
			} else {
				framework := v1.CoerceFramework(language, kind)
				template, err := templating.CompileTemplate(suite.Retries.Command)
				Expect(err).NotTo(HaveOccurred())
				Expect(targetedretries.SubstitutionsByFramework[framework].ValidateTemplate(template)).To(Succeed())
			}
		},
		Entry("xUnit", "xunit", ".net",
			`dotnet test --logger "xunit;LogFileName=xunit.xml" --results-directory "."`, "xunit.xml", "", " "),
		Entry("ExUnit", "exunit", "elixir", "mix test", "tmp/junit/*.xml", "test/**/*_test.exs", " "),
		Entry("Ginkgo", "ginkgo", "go", "ginkgo run --keep-going --json-report ./ginkgo.json ./...",
			"ginkgo.json", "", " "),
		Entry("go test", "go test", "go", "gotestsum --raw-command --jsonfile go-test.json -- go test ./... -json -count=1",
			"go-test.json", "", " "),
		Entry("Bun", "bun", "javascript", "bun test --reporter=junit --reporter-outfile bun.xml",
			"bun.xml", "**/*.test.ts", " "),
		Entry("JavaScript Cucumber", "cucumber", "javascript", "npx cucumber-js --format json:cucumber.json --format summary",
			"cucumber.json", "features/**/*.feature", " "),
		Entry("Cypress", "cypress", "javascript", "npx cypress run", "tmp/rwx/*.json", "cypress/**/*.cy.js", ","),
		Entry("Jest", "jest", "javascript", "npx jest --json --testLocationInResults --outputFile jest.json",
			"jest.json", "**/*.test.js", "|"),
		Entry("Karma", "karma", "javascript", "npx karma start --single-run", "tmp/karma.json", "", " "),
		Entry("Mocha", "mocha", "javascript",
			"npx mocha 'test/**/*.js' --reporter @rwx-research/mocha-multi-reporters "+
				"--reporter-options configFile=multi-reporters.json",
			"tmp/mocha.json", "test/**/*.js", " "),
		Entry("Playwright", "playwright", "javascript",
			"bash -c 'PLAYWRIGHT_JSON_OUTPUT_NAME=playwright.json npx playwright test --reporter=json'",
			"playwright.json", "tests/**/*.spec.js", " "),
		Entry("TestCafe", "testcafe", "javascript", "npx testcafe chrome:headless tests/ --reporter spec,json:testcafe.json",
			"testcafe.json", "tests/**/*.testcafe.ts", " "),
		Entry("Vitest", "vitest", "javascript",
			"npx vitest run --reporter=default --reporter=json --outputFile=./vitest.json",
			"vitest.json", "**/*.test.js", " "),
		Entry("PHPUnit", "phpunit", "php", "vendor/bin/phpunit --log-junit phpunit.xml tests/", "phpunit.xml", "", " "),
		Entry("pytest", "pytest", "python", "pytest --report-log=log.json", "log.json", "test/**/test_*.py", " "),
		Entry("unittest", "unittest", "python", "python -m xmlrunner -o .", "TEST-*.xml", "test/**/test_*.py", " "),
		Entry("Ruby Cucumber", "cucumber", "ruby", "bundle exec cucumber --format json --out cucumber.json --format pretty",
			"cucumber.json", "features/**/*.feature", " "),
		Entry("minitest", "minitest", "ruby", "bin/rails test", "tmp/reports/*.xml", "test/**/*_test.rb", " "),
		Entry("RSpec", "rspec", "ruby", "bundle exec rspec --format json --out rspec.json --format progress",
			"rspec.json", "spec/**/*_spec.rb", " "),
	)

	It("requires a language for ambiguous frameworks", func() {
		_, err := configure("run", "", "--framework", " CuCuMbEr ")
		configError, ok := errors.AsConfigurationError(err)
		Expect(ok).To(BeTrue())
		Expect(configError.Description()).To(ContainSubstring("JavaScript, Ruby"))
		Expect(configError.Resolution()).To(ContainSubstring("--language"))
	})

	It("matches frameworks and explicit languages case-insensitively", func() {
		cfg, err := configure("run", "", "--framework", " RsPeC ", "--language", " RuBy ")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.TestSuites["suite"].Results.Path).To(Equal("rspec.json"))
	})

	DescribeTable("preserves explicit overrides, including empty values", func(value string) {
		cfg, err := configure("run", "", "--framework", "rspec", "--language", "ruby",
			"--command", value, "--retry-command", value, "--partition-command", value,
			"--partition-globs", value, "--test-results", value, "--print-summary=false")
		Expect(err).NotTo(HaveOccurred())
		suite := cfg.TestSuites["suite"]
		Expect(suite.Command).To(Equal(value))
		Expect(suite.Retries.Command).To(Equal(value))
		Expect(suite.Partition.Command).To(Equal(value))
		Expect(suite.Partition.Globs).To(Equal([]string{value}))
		Expect(suite.Results.Path).To(Equal(value))
		Expect(suite.Output.PrintSummary).To(BeFalse())
	}, Entry("custom values", "custom"), Entry("empty values", ""))

	It("allows overriding framework-specific partition delimiters", func() {
		cfg, err := configure("run", "", "--framework", "cypress", "--partition-delimiter", " ")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.TestSuites["suite"].Partition.Delimiter).To(Equal(" "))
	})

	DescribeTable("does not apply defaults to configured suites", func(config string) {
		cfg, err := configure("run", config, "--framework", "rspec")
		Expect(err).NotTo(HaveOccurred())
		suite := cfg.TestSuites["suite"]
		Expect(suite.Command).To(BeEmpty())
		Expect(suite.Results.Language).To(BeEmpty())
		Expect(suite.Results.Path).To(BeEmpty())
		Expect(suite.Retries.Command).To(BeEmpty())
		Expect(suite.Partition.Command).To(BeEmpty())
		Expect(suite.Partition.Globs).To(BeEmpty())
		Expect(suite.Output.PrintSummary).To(BeFalse())
	},
		Entry("an empty mapping", "test-suites:\n  suite: {}\n"),
		Entry("a null suite", "test-suites:\n  suite:\n"),
		Entry("a partially configured suite", "test-suites:\n  suite:\n    retries:\n      attempts: 2\n"),
	)

	It("preserves a configured suite's values even when the framework flag changes", func() {
		cfg, err := configure("run", `test-suites:
  suite:
    command: custom run
    results:
      language: Ruby
      framework: RSpec
      path: custom.json
    retries:
      command: custom retry
    partition:
      command: custom partition
      globs: [custom/**/*]
`, "--framework", "cucumber")
		Expect(err).NotTo(HaveOccurred())
		suite := cfg.TestSuites["suite"]
		Expect(suite.Command).To(Equal("custom run"))
		Expect(suite.Results.Language).To(Equal("Ruby"))
		Expect(suite.Results.Path).To(Equal("custom.json"))
		Expect(suite.Retries.Command).To(Equal("custom retry"))
		Expect(suite.Partition.Command).To(Equal("custom partition"))
		Expect(suite.Partition.Globs).To(Equal([]string{"custom/**/*"}))
		Expect(suite.Output.PrintSummary).To(BeFalse())
	})

	It("applies defaults when only another suite is configured", func() {
		cfg, err := configure("run", "test-suites:\n  other: {}\n", "--framework", "rspec")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.TestSuites["suite"].Results.Path).To(Equal("rspec.json"))
		Expect(cfg.TestSuites["suite"].Output.PrintSummary).To(BeTrue())
		Expect(cfg.TestSuites["other"].Command).To(BeEmpty())
	})

	DescribeTable("leaves other commands and unknown frameworks alone", func(command string, flags []string) {
		cfg, err := configure(command, "", flags...)
		Expect(err).NotTo(HaveOccurred())
		suite := cfg.TestSuites["suite"]
		Expect(suite.Command).To(BeEmpty())
		Expect(suite.Results.Path).To(BeEmpty())
		Expect(suite.Output.PrintSummary).To(BeFalse())
	},
		Entry("parse", "results", []string{"--framework", "rspec"}),
		Entry("partition", "partition", []string{"--framework", "rspec"}),
		Entry("no framework", "run", []string{}),
		Entry("unknown framework", "run", []string{"--framework", "custom", "--language", "custom"}),
		Entry("explicit different language", "run", []string{"--framework", "rspec", "--language", "python"}),
	)
})
