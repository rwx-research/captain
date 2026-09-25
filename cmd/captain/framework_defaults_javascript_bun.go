package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptBunSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "bun test --reporter=junit --reporter-outfile bun.xml"
	suite.Results.Path = "bun.xml"
	suite.Retries.Command = "bun test '{{ file }}' --test-name-pattern '{{ testNamePattern }}' " +
		"--reporter=junit --reporter-outfile bun.xml"
	return suite
}
