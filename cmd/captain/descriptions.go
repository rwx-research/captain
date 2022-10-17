package main

// These constants hold the "long" description of a subcommand. These get printed when running `--help`, for example.
const (
	descriptionCaptain = `Captain provides client-side utilities related to build- and test-suites.

This CLI is a complementary component to the main WebUI at
https://captain.build.`

	descriptionUploadResults = `'captain upload results' will upload artifacts from various test runners, such
as JUnit or RSpec.

Example use:

	captain upload results --suite-name="JUnit" *.xml`
)
