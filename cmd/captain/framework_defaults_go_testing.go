package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultGoTestSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "gotestsum --raw-command --jsonfile go-test.json -- go test ./... -json -count=1"
	suite.Results.Path = "go-test.json"
	suite.Retries.Command = "gotestsum --raw-command --jsonfile go-test.json -- " +
		"go test {{ package }} -run '{{ run }}' -json -count=1"
	suite.Partition.DiscoveryCommand = "go list ./..."
	suite.Partition.Command = "gotestsum --raw-command --jsonfile go-test.json -- go test {{ testFiles }} -json -count=1"
	return suite
}
