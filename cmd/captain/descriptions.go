package main

// These constants hold the "long" description of a subcommand. These get printed when running `--help`, for example.
const (
	descriptionAddFlake = `'captain add flake' can be used to mark a test as flaky.

To select a test, specify the metadata that uniquely identifies a single test. For example:

        captain add flake --suite-id "example" --file "./test/controller_spec.rb" --description "My test"`

	descriptionAddQuarantine = `'captain add quarantine' can be used to quarantine a test.

To select a test, specify the metadata that uniquely identifies a single test. For example:

        captain add quarantine --suite-id "example" --file "./test/controller_spec.rb" --description "My test"`

	descriptionCaptain = `Captain provides client-side utilities related to build- and test-suites.

This CLI is a complementary component to the main WebUI at
https://captain.build.`

	descriptionUploadResults = `'captain upload results' will upload test results from various test runners, such
as JUnit or RSpec.

Example use:

	captain upload results --suite-id="JUnit" *.xml`

	descriptionUpdateResults = `'captain update results' will parse a test-results file and updates captain's
internal storage accordingly.

Example use:

	captain update results --suite-id="JUnit" *.xml`

	descriptionRemoveFlake = `'captain remove flake' can be used to remove a specific test for the list of flakes.
Effectively, this is the inverse of 'captain add flake'.`

	descriptionRemoveQuarantine = `'captain remove quarantine' can be used to remove a quarantine from a specific test.
Effectively, this is the inverse of 'captain add quarantine'.`

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

	descriptionQuarantine = `'captain quarantine' can be used to execute a test-suite and modify its exit code based on
quarantined tests.

You may be looking for 'captain run' instead. 'captain run' does everything 'captain quarantine' does, but it also
handles automatically retrying tests and optionally uploading results to Captain Cloud.

Example use:

	captain quarantine --suite-id="your-project-jest" --test-results "jest-result.json" -- jest`
)
