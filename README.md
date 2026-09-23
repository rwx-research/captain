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

For suites not defined in `.captain/config.yaml` (or `.yml`), `captain run` supplies
framework-specific commands, test result paths, retry and partition templates where
supported, and a printed summary:

```sh
captain run my-rspec-suite --framework rspec
captain run my-rspec-suite --framework rspec --retries 2
captain run my-cucumber-suite --framework cucumber --language ruby
```

Captain infers the language when a framework has only one supported language.
Cucumber requires `--language ruby` or `--language javascript`. Override individual
defaults with CLI flags, including `--print-summary=false`. Changing
`--test-results` does not rewrite the output path in command templates; override
the commands as well when changing where your framework writes results.

Defaults follow the [framework integration examples](https://www.rwx.com/docs/captain/test-frameworks).
You still need the documented test runners, reporter dependencies, and reporter
configuration. Command-controlled reports are written in the current directory;
paths controlled by reporter configuration retain the documented locations.
Retries and partitioning remain opt-in. **No framework defaults are applied to a
suite present in the configuration file**, even if its configuration is empty.

Partition defaults use framework discovery where supported:

- Go: `go list ./...` discovers packages.
- Ginkgo: `go list` discovers directories containing tests.
- Jest: `npx jest --listTests`, with selected files passed via `--runTestsByPath`.
- Vitest: `npx vitest list --filesOnly --json`, extracting file paths so named
  projects work too (requires a Vitest version supporting `--filesOnly`).
- Playwright: `npx playwright test --list --reporter=json`, extracting file paths.
- Bun has no list-only CLI mode; `find` discovers `*.test.ts` while pruning
  `node_modules` directories at every depth.

Other frameworks retain their existing partition defaults. For CLI-only suites,
`--partition-globs` replaces default discovery, and
`--partition-discovery-command` replaces default globs. Explicitly providing both
remains an error.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for information around our
development & contribution process.
