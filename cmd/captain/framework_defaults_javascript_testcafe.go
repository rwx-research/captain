package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptTestCafeSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "npx testcafe chrome:headless tests/ --reporter spec,json:testcafe.json"
	suite.Results.Path = "testcafe.json"
	suite.Retries.Command = "npx testcafe chrome:headless '{{ file }}' -T '{{ grep }}' --reporter spec,json:testcafe.json"
	suite.Partition.Command = "npx testcafe chrome:headless {{ testFiles }} --reporter spec,json:testcafe.json"
	suite.Partition.Globs = []string{"tests/**/*.testcafe.ts"}
	return suite
}
