package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultRubyRSpecSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "bundle exec rspec --format json --out rspec.json --format progress"
	suite.Results.Path = "rspec.json"
	suite.Retries.Command = suite.Command + " {{ tests }}"
	suite.Partition.Command = suite.Command + " {{ testFiles }}"
	suite.Partition.Globs = []string{"spec/**/*_spec.rb"}
	return suite
}
