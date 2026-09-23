package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptKarmaSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "npx karma start --single-run"
	suite.Results.Path = "tmp/karma.json"
	return suite
}
