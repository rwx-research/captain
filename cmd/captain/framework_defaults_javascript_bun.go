package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptBunSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "bun test --reporter=junit --reporter-outfile bun.xml"
	suite.Results.Path = "bun.xml"
	suite.Retries.Command = "bun test '{{ file }}' --test-name-pattern '{{ testNamePattern }}' " +
		"--reporter=junit --reporter-outfile bun.xml"
	suite.Partition.Command = "bun test {{ testFiles }} --reporter=junit --reporter-outfile bun.xml"
	suite.Partition.DiscoveryCommand = "find . -type d -name node_modules -prune -o -type f -name '*.test.ts' -print"
	return suite
}
