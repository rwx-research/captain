package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultRubyCucumberSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "bundle exec cucumber --format json --out cucumber.json --format pretty"
	suite.Results.Path = "cucumber.json"
	suite.Retries.Command = suite.Command + " {{ scenarios }}"
	suite.Partition.Command = suite.Command + " {{ testFiles }}"
	suite.Partition.Globs = []string{"features/**/*.feature"}
	return suite
}
