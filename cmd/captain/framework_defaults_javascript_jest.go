package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptJestSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "npx jest --json --testLocationInResults --outputFile jest.json"
	suite.Results.Path = "jest.json"
	suite.Retries.Command = "npx jest --testPathPattern '{{ testPathPattern }}' " +
		"--testNamePattern '{{ testNamePattern }}' --json --testLocationInResults --outputFile jest.json"
	suite.Partition.Command = "npx jest --testPathPattern {{ testFiles }} " +
		"--json --testLocationInResults --outputFile jest.json"
	suite.Partition.Delimiter = "|"
	suite.Partition.Globs = []string{"**/*.test.js"}
	return suite
}
