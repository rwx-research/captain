# Captain CLI

The Captain CLI is command-line app that integrates with the main Captain
platform at [https://captain.build](https://captain.build). Its main purpose
is to provide client-side utilities related to build- and test-suites. For
example, the CLI may provide the capability to make a test suite pass if the
only failed tests are identified as flaky & are quarantined in the Captain
WebUI. It will also provide the capability to upload test results to Captain.

For additional information, please refer to
[RFC 1](https://www.notion.so/rwx/RFC-1-Captain-CLI-architecture-82a164154abe48cdb92ad21050f63ef5)
on Notion.

## Development

### Dependencies

For local development, the following two dependencies are expected to be
installed on your system:

* [Go 1.19](https://go.dev)
* (optional) [golangci-lint v1.50](https://golangci-lint.run)

### Building, testing, and running

This project uses [Mage](https://magefile.org) as its build tool. To show a list
of available targets, run `go run ./tools/mage -l`:

```
Targets:
  build*    creates a production-ready build of the Captain CLI.
  clean     removes any generated artifacts from the repository.
  lint      runs the linter & performs static-analysis checks.
  run       runs the Captain CLI in a developer mode.
  test      executes the test-suite for the Captain-CLI.

* default target
```

If Mage is invoked without any arguments, the default target is being executed.
Otherwise, pass your desired target as the argument to Mage - multiple targets
can also be executed: `go run ./tools/mage lint test run`.

### Coding conventions

> Gofmt's style is no one's favorite, yet gofmt is everyone's favorite.

Try to follow the usual best practices around Go. Specifically, follow the
conventions from the core Go team:
[https://github.com/golang/go/wiki/CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments)

Our linters should handle common scenarios already, however they're not perfect.

### Tips

[Cobra](https://github.com/spf13/cobra-cli),
[Ginkgo](https://onsi.github.io/ginkgo/), and [Mage][https://magefile.org] can
all be installed as command-line applications. This is not strictly necessary,
but can be useful in a few situations. For example, both the `cobra` and
`ginkgo` CLIs allow generating common boilerplate code.
