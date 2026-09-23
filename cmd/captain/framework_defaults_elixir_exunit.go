package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultElixirExUnitSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "mix test"
	suite.Results.Path = "tmp/junit/*.xml"
	suite.Retries.Command = "mix test {{ tests }}"
	suite.Partition.Command = "mix test {{ testFiles }}"
	suite.Partition.Globs = []string{"test/**/*_test.exs"}
	return suite
}
