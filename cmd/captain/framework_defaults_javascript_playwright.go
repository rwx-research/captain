package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptPlaywrightSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "bash -c 'PLAYWRIGHT_JSON_OUTPUT_NAME=playwright.json npx playwright test --reporter=json'"
	suite.Results.Path = "playwright.json"
	suite.Retries.Command = `bash -c "PLAYWRIGHT_JSON_OUTPUT_NAME=playwright.json npx playwright test ` +
		`{{ tests }} --project '{{ project }}' --reporter=json"`
	suite.Partition.Command = "bash -c 'PLAYWRIGHT_JSON_OUTPUT_NAME=playwright.json " +
		"npx playwright test {{ testFiles }} --reporter=json'"
	suite.Partition.Globs = []string{"tests/**/*.spec.js"}
	return suite
}
