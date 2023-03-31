# Captain

Captain is an open source CLI that supercharges testing capabilities across 15
different testing frameworks. Use for faster and more reliable tests, and
happier and more productive engineers.

## Installation

You can download the latest release from the
[Releases page](https://github.com/rwx-research/captain-cli/releases) in this
repository.

Alternatively, if you have [Go](https://go.dev) installed, you can build the CLI
from source by cloning this repository and executing `go run ./tools/mage`.

## Usage

`captain --help` prints out an overview of the available commands. For more
detailed information, visit our documentation at
[https://www.rwx.com/docs/](https://www.rwx.com/docs/captain/cli-reference)

```
Usage:
  captain [command]

Available Commands:
  add         Adds a resource to captain
  partition   Partitions a test suite using historical file timings recorded by Captain
  quarantine  Execute a test-suite and modify its exit code based on quarantined tests
  remove      Removes a resource from captain
  run         Execute a build- or test-suite
  update      Updates a specific resource in captain

Flags:
      --config-file string   the config file for captain
  -h, --help                 help for captain
  -q, --quiet                disables most default output
      --suite-id string      the id of the test suite
  -v, --version              version for captain

Use "captain [command] --help" for more information about a command.
```

## Config file

We recommend placing a 
[config file](https://www.rwx.com/docs/captain/cli-configuration/config-yaml)
for captain at `.captain/config.yaml`. The file is optional, but it avoids
needing to pass a lot of flags with each invocation.

An example config file looks like this:

```yaml
cloud:
  disabled: true
test-suites:
  your-suite:
    command: bundle exec rspec
    output:
      reporters:
        rwx-v1-json: ./tmp/rwx.json
    results:
      language: Ruby
      framework: RSpec
      path: ./tmp/rspec.json
```

## Cloud mode

By default, Captain will work fully offline and stores test metadata in it's
`.captain` directory.

However, you can also use this CLI in conjunction with our Cloud offering at
[https://www.rwx.com](https://www.rwx.com/captain) for in-depth analytics of
your test suite over time, flaky test detection, and more.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for information around our
development & contribution process.
