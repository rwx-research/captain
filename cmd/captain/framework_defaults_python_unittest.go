package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultPythonUnitTestSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "python -m xmlrunner -o ."
	suite.Results.Path = "TEST-*.xml"
	suite.Retries.Command = suite.Command + " {{ tests }}"
	suite.Partition.Command = suite.Command + " {{ testFiles }}"
	suite.Partition.Globs = []string{"test/**/test_*.py"}
	return suite
}
