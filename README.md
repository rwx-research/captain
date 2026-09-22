# <img src="https://www.rwx.com/captain.svg" height="60" alt="captain">

:globe_with_meridians: [website](https://rwx.com/captain) &ensp; :heavy_multiplication_x: [@rwx_research](https://x.com/rwx_research) &ensp; :speech_balloon: [discord](https://discord.gg/h4ha5Cue7j) &ensp; :books: [documentation](https://www.rwx.com/docs/captain)

Captain is an open source CLI that can:
- detect and quarantine flaky tests
- automatically retry failed tests
- partition files for parallel execution
- generate comprehensive test failure summaries

See the [announcement blog post](https://www.rwx.com/blog/captain-1-10-generally-available-open-source-release)
and [documentation](https://www.rwx.com/docs/captain) to learn more.

## Getting Started

The Captain CLI is easy to integrate into a build process running on any CI platform.
See the [documentation on getting started](https://www.rwx.com/docs/captain).
We’re happy to help with any integrations. Say hello on [Discord](https://discord.gg/h4ha5Cue7j)
or reach out at [hello@rwx.com](mailto:hello@rwx.com)

## Partitioning Go packages and other identifiers

Use command discovery instead of file globs to partition opaque identifiers:

```sh
go test $(captain partition my-go-suite --index 0 --total 4 --discovery-command 'go list ./...')
```

For `captain run`, set `partition.discovery-command` in the suite configuration
(or `--partition-discovery-command`). The existing partition command's
`{{ testFiles }}` placeholder also accepts these identifiers:

```yaml
test-suites:
  my-go-suite:
    partition:
      discovery-command: go list ./...
      command: go test -json {{ testFiles }}
```

Discovery commands run with the same argument parsing as test commands; use
`sh -c '...'` explicitly for shell operators. Output is split on newlines by
default; `discovery-delimiter` (`--discovery-delimiter` for `partition`,
`--partition-discovery-delimiter` for `run`) changes the separator. Empty entries
and duplicate identifiers are ignored. Identifiers are matched exactly, without
path normalization or prefix trimming. Discovery commands and globs are mutually
exclusive.

Each suite uses a single timing manifest, with package names or other opaque
identifiers stored in the existing `file_path` field. No timing granularity
configuration is needed. The Go JSON parser records successful package elapsed
times, including package overhead and parallel execution, rather than summing
individual tests. Failed or incomplete
packages do not contribute timings, and targeted retries do not replace the
original run's package timings. Configure result capture as usual to record
these timings. Non-file timings also work with the local backend using the
suite's existing `timings.yaml` file.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for information around our
development & contribution process.
