//go:build integration

package integration_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(versionedPrefixForQuarantining()+"Cloud Mode Integration Tests", func() {
	BeforeEach(func() {
		Expect(os.Getenv("RWX_ACCESS_TOKEN_STAGING")).ToNot(BeEmpty(), "These integration tests require a valid RWX_ACCESS_TOKEN_STAGING")
	})

	withAndWithoutInheritedEnv(func(getEnv envGenerator, prefix string) {
		getEnvWithAccessToken := func() map[string]string {
			env := getEnv()
			env["CAPTAIN_HOST"] = "staging.cloud.rwx.com"
			env["RWX_ACCESS_TOKEN"] = os.Getenv("RWX_ACCESS_TOKEN_STAGING")
			return env
		}

		Describe("captain run", func() {
			Context("quarantining", func() {
				It("succeeds when all failures quarantined", func() {
					testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-quarantine.json", prefix)
					defer cleanup()

					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-quarantine-test",
							"--test-results", testResultsPath,
							"--fail-on-upload-error",
							"-c", "bash -c 'exit 2'",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(BeEmpty())
					})
					Expect(result.exitCode).To(Equal(0))
				})

				It("fails & passes through exit code when not all failures quarantined", func() {
					testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-quarantined-with-other-errors.json", prefix)
					defer cleanup()

					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-quarantine-test",
							"--test-results", testResultsPath,
							"--fail-on-upload-error",
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(Equal("Error: test suite exited with non-zero exit code"))
					})
					Expect(result.exitCode).To(Equal(123))
				})
			})

			Context("retries", func() {
				It("succeeds when all failures quarantined", func() {
					testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-quarantine.json", prefix)
					defer cleanup()

					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-quarantine-test",
							"--test-results", testResultsPath,
							"--fail-on-upload-error",
							"--retries", "1",
							"--retry-command", `echo "{{ tests }}"`,
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(ContainSubstring("The retry command of suite \"captain-cli-quarantine-test\" appears to be misconfigured."))
					})
					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("fails & passes through exit code on failure", func() {
					testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
					defer cleanup()

					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-functional-tests",
							"--test-results", testResultsPath,
							"--fail-on-upload-error",
							"--retries", "1",
							"--retry-command", `echo "{{ tests }}"`,
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(ContainSubstring("The retry command of suite \"captain-cli-functional-tests\" appears to be misconfigured."))
						Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
					})
					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(123))
				})

				It("fails on misconfigured retry command", func() {
					testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-quarantine.json", prefix)
					defer cleanup()

					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-quarantine-test",
							"--test-results", testResultsPath,
							"--fail-on-upload-error",
							"--fail-on-misconfigured-retry",
							"--retries", "1",
							"--retry-command", `echo "{{ tests }}"`,
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(ContainSubstring("The retry command of suite \"captain-cli-quarantine-test\" appears to be misconfigured."))
					})
					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(1))
				})

				It("succeeds with 'misconfigured-retry' flag when configured correctly", func() {
					testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
					defer cleanup()

					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-functional-tests",
							"--test-results", testResultsPath,
							"--fail-on-upload-error",
							"--fail-on-misconfigured-retry",
							"--retries", "1",
							"--retry-command", fmt.Sprintf("cp fixtures/integration-tests/rspec-passed-not-quarantined.json %s; echo \"{{ tests }}\"", testResultsPath),
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(BeEmpty())
					})
					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(0))
				})
			})

			Context("partition", func() {
				Context("without timings", func() {
					It("sets partition 1 correctly", func() {
						testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passing.json", prefix)
						defer cleanup()

						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"captain-cli-functional-tests",
								"--partition-command", "echo 'bin/rspec {{ testFiles }}'",
								"--test-results", testResultsPath,
								"--partition-globs", "fixtures/integration-tests/partition/x.rb",
								"--partition-globs", "fixtures/integration-tests/partition/y.rb",
								"--partition-globs", "fixtures/integration-tests/partition/z.rb",
								"--partition-index", "0",
								"--partition-total", "2",
								"-c", "bash -c 'exit 123'",
							},
							env: getEnvWithAccessToken(),
						})

						withoutBackwardsCompatibility(func() {
							Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
						})
						Expect(result.stdout).To(ContainSubstring("bin/rspec fixtures/integration-tests/partition/x.rb fixtures/integration-tests/partition/z.rb"))
						Expect(result.exitCode).To(Equal(0))
					})

					It("sets partition 2 correctly", func() {
						testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passing.json", prefix)
						defer cleanup()

						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"captain-cli-functional-tests",
								"--partition-command", "echo 'bin/rspec {{ testFiles }}'",
								"--test-results", testResultsPath,
								"--partition-globs", "fixtures/integration-tests/partition/x.rb",
								"--partition-globs", "fixtures/integration-tests/partition/y.rb",
								"--partition-globs", "fixtures/integration-tests/partition/z.rb",
								"--partition-index", "1",
								"--partition-total", "2",
								"-c", "bash -c 'exit 123'",
							},
							env: getEnvWithAccessToken(),
						})

						withoutBackwardsCompatibility(func() {
							Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
						})
						Expect(result.stdout).To(ContainSubstring("bin/rspec fixtures/integration-tests/partition/y.rb"))
						Expect(result.exitCode).To(Equal(0))
					})
				})

				Context("with timings", func() {
					BeforeEach(func() {
						// refresh timing information because they expire periodically
						Expect(runCaptain(captainArgs{
							args: []string{
								"upload", "results",
								"captain-cli-functional-tests",
								"fixtures/integration-tests/partition/rspec-partition.json",
							},
							env: getEnvWithAccessToken(),
						}).exitCode).To(Equal(0))
					})

					It("sets partition 1 correctly", func() {
						testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passing.json", prefix)
						defer cleanup()

						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"captain-cli-functional-tests",
								"--partition-command", "echo 'bin/rspec {{ testFiles }}'",
								"--test-results", testResultsPath,
								"--partition-globs", "fixtures/integration-tests/partition/*_spec.rb",
								"--partition-index", "0",
								"--partition-total", "2",
								"-c", "bash -c 'exit 123'",
							},
							env: getEnvWithAccessToken(),
						})

						withoutBackwardsCompatibility(func() {
							Expect(result.stderr).To(BeEmpty())
						})
						Expect(result.stdout).To(ContainSubstring("bin/rspec fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
						Expect(result.exitCode).To(Equal(0))
					})

					It("sets partition 2 correctly", func() {
						testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passing.json", prefix)
						defer cleanup()

						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"captain-cli-functional-tests",
								"--partition-command", "echo 'bin/rspec {{ testFiles }}'",
								"--test-results", testResultsPath,
								"--partition-globs", "fixtures/integration-tests/partition/*_spec.rb",
								"--partition-index", "1",
								"--partition-total", "2",
								"-c", "bash -c 'exit 123'",
							},
							env: getEnvWithAccessToken(),
						})

						withoutBackwardsCompatibility(func() {
							Expect(result.stderr).To(BeEmpty())
						})
						Expect(result.stdout).To(ContainSubstring("bin/rspec fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb"))
						Expect(result.exitCode).To(Equal(0))
					})
				})
			})
		})

		Describe("captain quarantine", func() {
			It("succeeds when all failures quarantined", func() {
				testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-quarantine.json", prefix)
				defer cleanup()

				result := runCaptain(captainArgs{
					args: []string{
						"quarantine",
						"captain-cli-quarantine-test",
						"--test-results", testResultsPath,
						"-c", "bash -c 'exit 2'",
					},
					env: getEnvWithAccessToken(),
				})

				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(BeEmpty())
				})
				Expect(result.exitCode).To(Equal(0))
			})

			It("fails & passes through exit code when not all failures quarantined", func() {
				testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-quarantined-with-other-errors.json", prefix)
				defer cleanup()

				result := runCaptain(captainArgs{
					args: []string{
						"quarantine",
						"captain-cli-quarantine-test",
						"--test-results", testResultsPath,
						"-c", "bash -c 'exit 123'",
					},
					env: getEnvWithAccessToken(),
				})

				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(Equal("Error: test suite exited with non-zero exit code"))
				})
				Expect(result.exitCode).To(Equal(123))
			})
		})

		Describe("captain partition", func() {
			Context("without timings", func() {
				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"captain-cli-functional-tests",
							"fixtures/integration-tests/partition/x.rb",
							"fixtures/integration-tests/partition/y.rb",
							"fixtures/integration-tests/partition/z.rb",
							"--index", "0",
							"--total", "2",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
					})
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/x.rb fixtures/integration-tests/partition/z.rb"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("sets partition 2 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"captain-cli-functional-tests",
							"fixtures/integration-tests/partition/x.rb",
							"fixtures/integration-tests/partition/y.rb",
							"fixtures/integration-tests/partition/z.rb",
							"--index", "1",
							"--total", "2",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
					})
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/y.rb"))
					Expect(result.exitCode).To(Equal(0))
				})
			})

			Context("with timings", func() {
				BeforeEach(func() {
					// refresh timing information because they expire periodically
					Expect(runCaptain(captainArgs{
						args: []string{
							"upload", "results",
							"captain-cli-functional-tests",
							"fixtures/integration-tests/partition/rspec-partition.json",
						},
						env: getEnvWithAccessToken(),
					}).exitCode).To(Equal(0))
				})

				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"captain-cli-functional-tests",
							"fixtures/integration-tests/partition/*_spec.rb",
							"--index", "0",
							"--total", "2",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(BeEmpty())
					})
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("sets partition 2 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"captain-cli-functional-tests",
							"fixtures/integration-tests/partition/*_spec.rb",
							"--index", "1",
							"--total", "2",
						},
						env: getEnvWithAccessToken(),
					})

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(BeEmpty())
					})
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})
			})
		})

		Describe("captain upload", func() {
			It("short circuits when there's nothing to upload", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"upload", "results",
						"captain-cli-functional-tests",
						"nonexistingfile.json",
					},
					env: getEnvWithAccessToken(),
				})

				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).To(BeEmpty())
				})
				Expect(result.exitCode).To(Equal(0))
			})
		})
	})
})
