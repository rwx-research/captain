package main

// These constants hold the "long" description of a subcommand. These get printed when running `--help`, for example.
const (
	descriptionCaptain = `Captain provides client-side utilities related to build- and test-suites.

This CLI is a complementary component to the main WebUI at
https://captain.build.`

	descriptionUploadResults = `'captain upload results' will upload test results from various test runners, such
as JUnit or RSpec.

Example use:

	captain upload results --suite-id="JUnit" *.xml`

	descriptionRun = `'captain run' can be used to execute a build- or test-suite and optionally upload the resulting
artifacts.

Example use:

	captain run --suite-id="your-project-rake" -- bundle exec rake

	captain run --suite-id="your-project-jest" --test-results "jest-result.json" -- jest`

	descriptionPartition = `'captain partition' can be used to split up your test suite by test file, leveraging test
file timings recorded in captain.

Example use:

	bundle exec rspec $(captain partition spec/**/*_spec.rb --suide-id your-project-rspec --index 0 --total 2)
	bundle exec rspec $(captain partition spec/**/*_spec.rb --suide-id your-project-rspec --index 1 --total 2)`
)
