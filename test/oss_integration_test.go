//go:build integration

package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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
				var suiteId string
				BeforeEach(func() {
					// create a random suite ID so that concurrent tests don't try to read / write from the same timings files
					suiteUUID, err := uuid.NewRandom()
					if err != nil {
						Expect(err).ToNot(HaveOccurred())
					}

					suiteId = suiteUUID.String()

					_ = runCaptain(captainArgs{
						args: []string{
							"update", "results",
							"fixtures/integration-tests/partition/rspec-partition.json",
							"--suite-id", suiteId,
						},
						env: getEnvWithoutAccessToken(),
					})
				})

				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							"fixtures/integration-tests/partition/*_spec.rb",
							"--suite-id", suiteId,
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
							"--suite-id", suiteId,
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

				It("succeeds when all failures quarantined", func() {
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

					Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'")) // indicative of a retry
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

			Context("quarantining", func() {
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

					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
					Expect(result.exitCode).To(Equal(123))
				})
			})

			Context("with abq", func() {
				It("runs with ABQ_SET_EXIT_CODE=false when ABQ_SET_EXIT_CODE is unset", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'echo exit_code=$ABQ_SET_EXIT_CODE'",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stdout).To(HavePrefix("exit_code=false"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("runs with ABQ_SET_EXIT_CODE=false when ABQ_SET_EXIT_CODE is already set", func() {
					env := getEnvWithoutAccessToken()
					env["ABQ_SET_EXIT_CODE"] = "1234"
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'echo exit_code=$ABQ_SET_EXIT_CODE'",
						},
						env: env,
					})

					Expect(result.stdout).To(HavePrefix("exit_code=false"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("runs with new ABQ_STATE_FILE path when ABQ_STATE_FILE is unset", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'echo state_file=$ABQ_STATE_FILE'",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stdout).To(HavePrefix("state_file=/tmp/captain-abq-"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("runs with previously set ABQ_STATE_FILE path when ABQ_STATE_FILE is set", func() {
					env := getEnvWithoutAccessToken()
					env["ABQ_STATE_FILE"] = "/tmp/functional-abq-1234.json"
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'echo state_file=$ABQ_STATE_FILE'",
						},
						env: env,
					})

					Expect(result.stdout).To(HavePrefix("state_file=/tmp/functional-abq-1234.json"))
					Expect(result.exitCode).To(Equal(0))
				})
			})
		})

		DescribeTable("captain [add|remove]",
			func(resource string) {
				read := func(path string) string {
					data, err := ioutil.ReadFile(path)
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}

				suiteUUID, err := uuid.NewRandom()
				Expect(err).ToNot(HaveOccurred())
				suiteID := suiteUUID.String()

				By("adding")
				result := runCaptain(captainArgs{
					args: []string{
						"add", resource,
						"--suite-id", suiteID,
						"--file", "./x.rb",
						"--description", "is flaky",
					},
					env: getEnvWithoutAccessToken(),
				})
				Expect(result.exitCode).To(Equal(0))

				lines := strings.Split(read(fmt.Sprintf(".captain/%s/%ss.yaml", suiteID, resource)), "\n")
				Expect(lines).To(HaveLen(3))
				Expect(lines[0]).To(Equal("- file: ./x.rb"))
				Expect(lines[1]).To(Equal("  description: is flaky"))
				Expect(lines[2]).To(Equal(""))

				By("removing")
				result = runCaptain(captainArgs{
					args: []string{
						"remove", resource,
						"--suite-id", suiteID,
						"--file", "./x.rb",
						"--description", "is flaky",
					},
					env: getEnvWithoutAccessToken(),
				})
				Expect(result.exitCode).To(Equal(0))

				lines = strings.Split(read(fmt.Sprintf(".captain/%s/%ss.yaml", suiteID, resource)), "\n")
				Expect(lines).To(HaveLen(2))
				Expect(lines[0]).To(Equal("[]"))
				Expect(lines[1]).To(Equal(""))
			},
			Entry("for quarantines", "quarantine"),
			Entry("for flakes", "flake"),
		)
	})
})
