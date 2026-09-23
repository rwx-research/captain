package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptCypressSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "npx cypress run"
	suite.Results.Path = "tmp/rwx/*.json"
	suite.Retries.Command = "npx cypress run --spec '{{ spec }}' --env {{ grep }}"
	suite.Partition.Command = "npx cypress run --spec {{ testFiles }}"
	suite.Partition.Delimiter = ","
	suite.Partition.Globs = []string{"cypress/**/*.cy.js"}
	return suite
}
