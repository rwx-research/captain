package main

import "github.com/rwx-research/captain-cli/internal/cli"

func defaultGoGinkgoSuite() cli.SuiteConfig {
	var suite cli.SuiteConfig
	suite.Command = "ginkgo run --keep-going --json-report ./ginkgo.json ./..."
	suite.Results.Path = "ginkgo.json"
	suite.Retries.Command = "ginkgo run {{ tests }} --keep-going --json-report ./ginkgo.json ./..."
	suite.Partition.DiscoveryCommand = "go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.Dir}}{{end}}' ./..."
	suite.Partition.Command = "ginkgo run --keep-going --json-report ./ginkgo.json {{ testFiles }}"
	return suite
}
