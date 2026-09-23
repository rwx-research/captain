package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultRubyMinitestSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "bin/rails test"
	suite.Results.Path = "tmp/reports/*.xml"
	suite.Retries.Command = "bin/rails test {{ tests }}"
	suite.Partition.Command = "bin/rails test {{ testFiles }}"
	suite.Partition.Globs = []string{"test/**/*_test.rb"}
	return suite
}
