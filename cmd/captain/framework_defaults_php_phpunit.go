package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultPHPUnitSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "vendor/bin/phpunit --log-junit phpunit.xml tests/"
	suite.Results.Path = "phpunit.xml"
	suite.Retries.Command = "vendor/bin/phpunit --log-junit phpunit.xml --filter '{{ filter }}' '{{ file }}'"
	return suite
}
