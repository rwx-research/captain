//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rwx-research/captain-cli"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OSS mode Integration Tests", func() {
	withAndWithoutInheritedEnv(func(getEnv envGenerator, prefix string) {
		getEnvWithoutAccessToken := func() map[string]string {
			// when env is inherited, RWX_ACCESS_TOKEN is present
			env := getEnv()
			delete(env, "RWX_ACCESS_TOKEN")
			return env
		}
		Describe("captain --version", func() {
			It("renders the version I expect", func() {
				result := runCaptain(captainArgs{
					args: []string{"--version"},
					env:  getEnvWithoutAccessToken(),
				})

				Expect(result.exitCode).To(Equal(0))
				Expect(result.stdout).To(Equal(captain.Version))
				Expect(result.stdout).To(MatchRegexp(`^v\d+\.\d+\.\d+$`))
				Expect(result.stderr).To(BeEmpty())
			})
		})

		Describe("captain partition", func() {
			Context("without timings", func() {
				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/x.rb",
							"fixtures/integration-tests/partition/y.rb",
							"fixtures/integration-tests/partition/z.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "0",
							"--total", "2",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/x.rb fixtures/integration-tests/partition/z.rb"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("sets partition 2 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/x.rb",
							"fixtures/integration-tests/partition/y.rb",
							"fixtures/integration-tests/partition/z.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "1",
							"--total", "2",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/y.rb"))
					Expect(result.exitCode).To(Equal(0))
				})
			})

			Context("with timings", func() {
				BeforeEach(func() {
					// 1. captain upload results test/fixtures/integration-tests/partition/rspec-partition.json --suite-id captain-cli-functional-tests
					_ = runCaptain(captainArgs{
						args: []string{
							"update", "results",
							"fixtures/integration-tests/partition/rspec-partition.json",
							"--suite-id", "captain-cli-functional-tests",
						},
						env: getEnvWithoutAccessToken(),
					})
				})

				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/*_spec.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "0",
							"--total", "2",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("sets partition 2 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/*_spec.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "1",
							"--total", "2",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})
			})

			Context("globbing", func() {
				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/**/*_spec.rb",
							"--suite-id", "captain-cli-functional-tests",
							"--index", "0",
							"--total", "1",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})
			})
		})

		Describe("captain run", func() {
			It("fails & passes through exit code on failure when results files is missing", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"--suite-id", "captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/does-not-exist.json",
						"-c", "bash -c 'exit 123'",
					},
					env: getEnvWithoutAccessToken(),
				})

				// Stderr being empty isn't ideal. See: https://github.com/rwx-research/captain-cli/issues/243
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.stdout).To(BeEmpty())
				Expect(result.exitCode).To(Equal(123))
			})

			It("fails & passes through exit code on failure", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"--suite-id", "captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
						"-c", "bash -c 'exit 123'",
					},
					env: getEnvWithoutAccessToken(),
				})

				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.exitCode).To(Equal(123))
			})

			It("allows using the -- command specification", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"--suite-id", "captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
						"--", "bash", "-c", "exit 123",
					},
					env: getEnvWithoutAccessToken(),
				})

				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.exitCode).To(Equal(123))
			})

			It("allows combining the --command specification with the -- specification", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"--suite-id", "captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
						"-c", "bash",
						"--", "-c", "exit 123",
					},
					env: getEnvWithoutAccessToken(),
				})

				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.exitCode).To(Equal(123))
			})

			It("accepts positional arguments", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-passed.json",
						"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
					},
					env: getEnvWithoutAccessToken(),
				})

				Expect(result.exitCode).To(Equal(0))
			})

			Context("command output passthrough", func() {
				It("passes through output of test command", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-functional-tests",
							"--test-results", "fixtures/integration-tests/rspec-passed.json",
							"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stderr).To(HaveSuffix("def")) // prefix by a bunch of warnings about captain dir not existing
					Expect(result.stdout).To(HavePrefix("abc\nghi"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("passes through output of test command in the correct order", func() {
					cmd := captainCmd(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-functional-test",
							"--test-results", "fixtures/integration-tests/rspec-passed.json",
							"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
						},
						env: getEnvWithoutAccessToken(),
					})

					combinedOutputBytes, err := cmd.CombinedOutput()
					Expect(err).ToNot(HaveOccurred())

					Expect(string(combinedOutputBytes)).To(ContainSubstring("abc\ndef\nghi\n"))
				})
			})

			Context("retries", func() {
				var _symlinkDestPath string
				var _symlinkSrcPath string

				// retry tests delete test results between retries.
				// this function ensures a symlink exists to the test results file
				// that can be freely removed
				// the symlink will be resuscitated after the test in the AfterEach
				symlinkToNewPath := func(srcPath string, prefix string) string {
					var err error
					_symlinkDestPath = fmt.Sprintf("fixtures/integration-tests/retries/%s-%s", prefix, filepath.Base(srcPath))
					_symlinkSrcPath = fmt.Sprintf("../%s", filepath.Base(srcPath))
					Expect(err).ToNot(HaveOccurred())

					os.Symlink(_symlinkSrcPath, _symlinkDestPath)
					return _symlinkDestPath
				}

				AfterEach(func() {
					os.Symlink(_symlinkSrcPath, _symlinkDestPath)
				})

				PIt("succeeds when all failures quarantined", func() {
					Skip("quarantining doesn't yet work")
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-quarantine-test",
							"--test-results", symlinkToNewPath("fixtures/integration-tests/rspec-quarantine.json", prefix),
							"--retries", "1",
							"--retry-command", `echo "{{ tests }}"`,
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("fails & passes through exit code on failure", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-functional-tests",
							"--test-results", symlinkToNewPath("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix),
							"--retries", "1",
							"--retry-command", `echo "{{ tests }}"`,
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(123))
				})

				It("fails & passes through exit code on failure when configured via captain config", func() {
					_ = symlinkToNewPath("fixtures/integration-tests/rspec-failed-not-quarantined.json", fmt.Sprintf("with-config-%s", prefix))
					result := runCaptain(captainArgs{
						args: []string{
							"run", fmt.Sprintf("%s-quarantined-retries-with-config", prefix),
							"--config-file", "fixtures/integration-tests/retry-config.yaml",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
					Expect(result.exitCode).To(Equal(123))
				})
			})

			PContext("quarantining (oss captain add quarantine not yet implemented)", func() {
				BeforeEach(func() {
					_ = runCaptain(captainArgs{
						args: []string{
							"add", "quarantine",
							"--suite-id", "captain-cli-quarantine-test",
							"--file", "./x.rb",
							"--description", `"is flaky"`,
						},
						env: getEnvWithoutAccessToken(),
					})
				})
				It("succeeds when all failures quarantined", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-quarantine-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"-c", "bash -c 'exit 2'",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.exitCode).To(Equal(0))
				})

				It("fails & passes through exit code when not all failures quarantined", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-quarantine-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantined-with-other-errors.json",
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stderr).To(Equal("Error: test suite exited with non-zero exit code"))
					Expect(result.exitCode).To(Equal(123))
				})
			})
		})
	})
})
