package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultJavaScriptMochaSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	const reporter = " --reporter @rwx-research/mocha-multi-reporters --reporter-options configFile=multi-reporters.json"
	suite.Command = "npx mocha 'test/**/*.js'" + reporter
	suite.Results.Path = "tmp/mocha.json"
	suite.Retries.Command = "npx mocha '{{ file }}' --grep '{{ grep }}'" + reporter
	suite.Partition.Command = "npx mocha {{ testFiles }}" + reporter
	suite.Partition.Globs = []string{"test/**/*.js"}
	return suite
}
