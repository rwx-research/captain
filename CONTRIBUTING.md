# Contributing to Captain

Captain is an open-source project and we welcome any contributions from other
developers interested in test automation.

## Filing Issues

When opening a new GitHub issue, make sure to include any relevant information,
such as:

* What version of Captain are you using (see `captain --version`)?
* What system / CI environment are you running Captain on?
* What did you do?
* What did you expect to see?
* What did you see instead?

## Contributing Code

We use GitHub pull requests for code contributions and reviews.

Our CI system will run tests & our linting setup against new pull requests, but
to shorten your feedback cycle, we would appreciate if
`go run ./tools/mage lint` and `go run ./tools/mage unittest` pass before
opening a PR.

### Development setup

You should not need any dependencies outside of Go to work on Captain.

We use [Mage](https://magefile.org) as the build tool for our project. To show
a list of available targets, run `go run ./tools/mage -l`:

```
Targets:
  all                cleans output, builds, tests, and lints.
  build*             builds the Captain CLI
  clean              removes any generated artifacts from the repository.
  integrationTest    Test executes the test-suite for the Captain-CLI.
  lint               runs the linter & performs static-analysis checks.
  lintFix            Applies lint checks and fixes any issues.
  test               
  unitTest           Test executes the test-suite for the Captain-CLI.

* default target
```

Mage can also be installed as command-line applications. This is not strictly
necessary, but is a bit nicer to use.

Note: Some tests use snapshot tests. If you need to update the snapshot tests,
run `UPDATE_SNAPSHOTS=true mage test`. Don't forget to check the diffs!

### Configuration

To simulate various CI environment, source the "env" files found in the `test/`
subdirectory. For example, to trigger GitHub specific behavior in Captain,
source `test/.env.github.actions`.

### Debugging

Besides the `--debug` flag, some useful options during development are:

* `--insecure` to disable TLS for the API
* `CAPTAIN_HOST` to route API traffic to a different host.
* `captain parse <file>` to parse a test result and return the internal
  representation as JSON.
