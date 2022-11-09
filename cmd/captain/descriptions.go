package main

// These constants hold the "long" description of a subcommand. These get printed when running `--help`, for example.
const (
	descriptionCaptain = `Captain provides client-side utilities related to build- and test-suites.

This CLI is a complementary component to the main WebUI at
https://captain.build.`

	descriptionUploadResults = `'captain upload results' will upload test results from various test runners, such
as JUnit or RSpec.

Example use:

	captain upload results --suite-name="JUnit" *.xml`

	descriptionRun = `'captain run' can be used to execute a build- or test-suite and optionally upload the resulting
artifacts.

Example use:

	captain run --suite-name="Rake" -- bundle exec rake

	captain run --suite-name="Jest" --test-results "jest-result.json" -- jest`
)
