package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptVitestSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "npx vitest run --reporter=default --reporter=json --outputFile=./vitest.json"
	suite.Results.Path = "vitest.json"
	suite.Retries.Command = "npx vitest run '{{ file }}' --testNamePattern '{{ testNamePattern }}' " +
		"--reporter=default --reporter=json --outputFile=./vitest.json"
	suite.Partition.Command = "npx vitest run {{ testFiles }} " +
		"--reporter=default --reporter=json --outputFile=./vitest.json"
	suite.Partition.Globs = []string{"**/*.test.js"}
	return suite
}
