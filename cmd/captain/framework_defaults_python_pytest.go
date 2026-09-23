package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultPythonPytestSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "pytest --report-log=log.json"
	suite.Results.Path = "log.json"
	suite.Retries.Command = suite.Command + " {{ tests }}"
	suite.Partition.Command = suite.Command + " {{ testFiles }}"
	suite.Partition.Globs = []string{"test/**/test_*.py"}
	return suite
}
