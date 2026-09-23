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
	suite.Partition.DiscoveryCommand = `node -e 'const {execFileSync} = require("node:child_process"); ` +
		`const env = {...process.env, PLAYWRIGHT_JSON_OUTPUT_FILE: "", PLAYWRIGHT_JSON_OUTPUT_NAME: ""}; ` +
		`const report = JSON.parse(execFileSync("npx", ["playwright", "test", "--list", "--reporter=json"], ` +
		`{encoding: "utf8", env})); ` +
		`for (const suite of report.suites) console.log(require("node:path").resolve(report.config.rootDir, suite.file));'`
	return suite
}
