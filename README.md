# Captain CLI

The Captain CLI is command-line app that integrates with the main Captain
platform at [https://captain.build](https://captain.build). Its main purpose
is to provide client-side utilities related to build- and test-suites. For
example, the CLI may provide the capability to make a test suite pass if the
only failed tests are identified as flaky & are quarantined in the Captain
WebUI. It will also provide the capability to upload test results to Captain.

For additional information, please refer to
[RFC 1](https://www.notion.so/rwx/RFC-1-Captain-CLI-architecture-82a164154abe48cdb92ad21050f63ef5)
and [RFC 3](https://www.notion.so/rwx/RFC-3-Captain-CLI-design-7c98d1dbf4244b0eb8960cc98c103f73)
on Notion.

## Usage

```
Captain provides client-side utilities related to build- and test-suites.

This CLI is a complementary component to the main WebUI at
https://captain.build.

Usage:
  captain [flags]
  captain [command]

Available Commands:
  help        Help about any command
  run         Execute a build- or test-suite
  upload      Upload a resource to Captain

Flags:
      --github-job-matrix string   the JSON encoded job-matrix from Github
      --github-job-name string     the name of the current Github Job
  -h, --help                       help for captain
      --suite-name string          the name of the build- or test-suite

Use "captain [command] --help" for more information about a command.
```

Currently, the CLI only supports uploading test results from Github Actions.
For example, the following command will upload all JSON files to Captain when
run in Github Actions:

```
./captain upload results --suite-name RSpec **/*.json
```

### Hidden options

To aid development, the following commands & flags are available, but hidden in
the documentation:

* `--insecure` disables TLS on the API. Useful for running against localhost.
* `parse` (as in `captain parse <file>`) attempts to parse one or more files and
  returns a normalized version of the test results.

## Development

### Setup

#### Dependencies

For local development, the following two dependencies are expected to be
installed on your system:

* [Go 1.19](https://go.dev)
* (optional) [golangci-lint v1.50](https://golangci-lint.run)

#### Environment

As the CLI is currently expected to be run from inside Github Actions, certain
environment variables are expected to be present. These can be found under
`test/.env.github.actions`.

### Building, testing, and running

This project uses [Mage](https://magefile.org) as its build tool. To show a list
of available targets, run `go run ./tools/mage -l`:

```
Targets:
  build*    builds the Captain CLI
  clean     removes any generated artifacts from the repository.
  lint      runs the linter & performs static-analysis checks.
  test      executes the test-suite for the Captain-CLI.

* default target
```

### Configuration

The following environment variables can / should be used to configure the CLI:

```
CAPTAIN_HOST        # The host or host:port combination for the Captain API
CAPTAIN_TOKEN       # The Captain API token
GITHUB_JOB          # The Job ID of a Github Actions Job
GITHUB_RUN_ATTEMPT  # A unique number for each attempt of a particular worklow
                    # on Github Actions
GITHUB_RUN_ID       # A unique number for each workflow run on Github Acations
GITHUB_REPOSITORY   # The owner & repository pair of the GitHub repository for
                    # Github Actions
```

### Coding conventions

> Gofmt's style is no one's favorite, yet gofmt is everyone's favorite.

Try to follow the usual best practices around Go. Specifically, follow the
conventions from the core Go team:
[https://github.com/golang/go/wiki/CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments)

Our linters should handle common scenarios already, however they're not perfect.

### Tips

[Cobra](https://github.com/spf13/cobra-cli),
[Ginkgo](https://onsi.github.io/ginkgo/), and [Mage](https://magefile.org) can
all be installed as command-line applications. This is not strictly necessary,
but can be useful in a few situations. For example, both the `cobra` and
`ginkgo` CLIs allow generating common boilerplate code.
