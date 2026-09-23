package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultDotNetxUnitSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = `dotnet test --logger "xunit;LogFileName=xunit.xml" --results-directory "."`
	suite.Results.Path = "xunit.xml"
	suite.Retries.Command = suite.Command + ` --filter '{{ filter }}'`
	return suite
}
