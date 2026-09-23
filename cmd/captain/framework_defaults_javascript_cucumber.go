package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptCucumberSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "npx cucumber-js --format json:cucumber.json --format summary"
	suite.Results.Path = "cucumber.json"
	suite.Retries.Command = suite.Command + " {{ scenarios }}"
	suite.Partition.Command = suite.Command + " {{ testFiles }}"
	suite.Partition.Globs = []string{"features/**/*.feature"}
	return suite
}
