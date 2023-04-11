//go:build integration

package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/rwx-research/captain-cli"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rwx-research/captain-cli/internal/cli"
)

var _ = Describe("OSS mode Integration Tests", func() {
	randomSuiteId := func() string {
		// create a random suite ID so that concurrent tests don't try to read / write from the same timings files
		suiteUUID, err := uuid.NewRandom()
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		return suiteUUID.String()
	}

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
				Expect(result.stdout).To(MatchRegexp(`^v\d+\.\d+\.\d+-?`))
				Expect(result.stderr).To(BeEmpty())
			})
		})

		Describe("captain partition", func() {
			Context("without timings", func() {
				Context("with a delimiter", func() {
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
								"--delimiter", ",",
							},
							env: getEnvWithoutAccessToken(),
						})

						Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
						Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/x.rb,fixtures/integration-tests/partition/z.rb"))
						Expect(result.exitCode).To(Equal(0))
					})

					It("sets partition 1 correctly when delimiter is set via env var", func() {
						env := getEnvWithoutAccessToken()
						env["CAPTAIN_DELIMITER"] = ","

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
							env: env,
						})

						Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
						Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/x.rb,fixtures/integration-tests/partition/z.rb"))
						Expect(result.exitCode).To(Equal(0))
					})
				})

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
							"captain-cli-functional-tests",
							"--index", "1",
							"--total", "2",
							"fixtures/integration-tests/partition/x.rb",
							"fixtures/integration-tests/partition/y.rb",
							"fixtures/integration-tests/partition/z.rb",
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
					suiteId = randomSuiteId()

					Expect(runCaptain(captainArgs{
						args: []string{
							"update", "results",
							suiteId,
							"fixtures/integration-tests/partition/rspec-partition.json",
						},
						env: getEnvWithoutAccessToken(),
					}).exitCode).To(Equal(0))
				})

				It("sets partition 1 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							suiteId,
							"fixtures/integration-tests/partition/*_spec.rb",
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
							suiteId,
							"fixtures/integration-tests/partition/*_spec.rb",
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
							"captain-cli-functional-tests",
							"fixtures/integration-tests/**/*_spec.rb",
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
						"captain-cli-functional-tests",
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
						"captain-cli-functional-tests",
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
						"captain-cli-functional-tests",
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
						"captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
						"-c", "bash",
						"--", "-c", "exit 123",
					},
					env: getEnvWithoutAccessToken(),
				})

				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.exitCode).To(Equal(123))
			})

			It("accepts --suite-id argument", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"--suite-id", "captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-passed.json",
						"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
					},
					env: getEnvWithoutAccessToken(),
				})

				Expect(result.exitCode).To(Equal(0))
			})

			It("produces markdown-summary reports via config file", func() {
				tmp, err := os.MkdirTemp("", "*")
				Expect(err).NotTo(HaveOccurred())

				os.Remove(filepath.Join(tmp, "markdown.md"))

				cfg := cli.ConfigFile{
					TestSuites: map[string]cli.SuiteConfig{
						"captain-cli-functional-tests": {
							Command:           "bash -c 'exit 123'",
							FailOnUploadError: true,
							Output: cli.SuiteConfigOutput{
								Reporters: map[string]string{"markdown-summary": filepath.Join(tmp, "markdown.md")},
							},
							Results: cli.SuiteConfigResults{
								Path: "fixtures/integration-tests/rspec-failed-not-quarantined.json",
							},
						},
					},
				}

				withCaptainConfig(cfg, tmp, func(configPath string) {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-functional-tests",
							"--config-file", configPath,
						},
						env: getEnvWithoutAccessToken(),
					})

					_, err = os.Stat(filepath.Join(tmp, "markdown.md"))
					Expect(err).NotTo(HaveOccurred())

					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
					Expect(result.exitCode).To(Equal(123))
				})
			})

			It("produces markdown-summary reports via CLI flag", func() {
				tmp, err := os.MkdirTemp("", "*")
				Expect(err).NotTo(HaveOccurred())

				os.Remove(filepath.Join(tmp, "markdown.md"))

				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
						"--fail-on-upload-error",
						"--reporter", fmt.Sprintf("markdown-summary=%v", filepath.Join(tmp, "markdown.md")),
						"-c", "bash -c 'exit 123'",
					},
					env: getEnvWithoutAccessToken(),
				})

				_, err = os.Stat(filepath.Join(tmp, "markdown.md"))
				Expect(err).NotTo(HaveOccurred())

				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.exitCode).To(Equal(123))
			})

			It("produces rwx-v1-json reports via config file", func() {
				tmp, err := os.MkdirTemp("", "*")
				Expect(err).NotTo(HaveOccurred())

				os.Remove(filepath.Join(tmp, "rwx-v1.json"))

				cfg := cli.ConfigFile{
					TestSuites: map[string]cli.SuiteConfig{
						"captain-cli-functional-tests": {
							Command:           "bash -c 'exit 123'",
							FailOnUploadError: true,
							Output: cli.SuiteConfigOutput{
								Reporters: map[string]string{"rwx-v1-json": filepath.Join(tmp, "rwx-v1.json")},
							},
							Results: cli.SuiteConfigResults{
								Path: "fixtures/integration-tests/rspec-failed-not-quarantined.json",
							},
						},
					},
				}

				withCaptainConfig(cfg, tmp, func(configPath string) {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-functional-tests",
							"--config-file", configPath,
						},
						env: getEnvWithoutAccessToken(),
					})

					_, err = os.Stat(filepath.Join(tmp, "rwx-v1.json"))
					Expect(err).NotTo(HaveOccurred())

					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
					Expect(result.exitCode).To(Equal(123))
				})
			})

			It("produces rwx-v1-json reports via CLI flag", func() {
				tmp, err := os.MkdirTemp("", "*")
				Expect(err).NotTo(HaveOccurred())

				os.Remove(filepath.Join(tmp, "rwx-v1.json"))

				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
						"--fail-on-upload-error",
						"--reporter", fmt.Sprintf("rwx-v1-json=%v", filepath.Join(tmp, "rwx-v1.json")),
						"-c", "bash -c 'exit 123'",
					},
					env: getEnvWithoutAccessToken(),
				})

				_, err = os.Stat(filepath.Join(tmp, "rwx-v1.json"))
				Expect(err).NotTo(HaveOccurred())

				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.exitCode).To(Equal(123))
			})

			It("produces junit-xml reports via config file", func() {
				tmp, err := os.MkdirTemp("", "*")
				Expect(err).NotTo(HaveOccurred())

				os.Remove(filepath.Join(tmp, "junit.xml"))

				cfg := cli.ConfigFile{
					TestSuites: map[string]cli.SuiteConfig{
						"captain-cli-functional-tests": {
							Command:           "bash -c 'exit 123'",
							FailOnUploadError: true,
							Output: cli.SuiteConfigOutput{
								Reporters: map[string]string{"junit-xml": filepath.Join(tmp, "junit.xml")},
							},
							Results: cli.SuiteConfigResults{
								Path: "fixtures/integration-tests/rspec-failed-not-quarantined.json",
							},
						},
					},
				}

				withCaptainConfig(cfg, tmp, func(configPath string) {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-functional-tests",
							"--config-file", configPath,
						},
						env: getEnvWithoutAccessToken(),
					})

					_, err = os.Stat(filepath.Join(tmp, "junit.xml"))
					Expect(err).NotTo(HaveOccurred())

					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
					Expect(result.exitCode).To(Equal(123))
				})
			})

			It("produces junit-xml reports via CLI flag", func() {
				tmp, err := os.MkdirTemp("", "*")
				Expect(err).NotTo(HaveOccurred())

				os.Remove(filepath.Join(tmp, "junit.xml"))

				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
						"--fail-on-upload-error",
						"--reporter", fmt.Sprintf("junit-xml=%v", filepath.Join(tmp, "junit.xml")),
						"-c", "bash -c 'exit 123'",
					},
					env: getEnvWithoutAccessToken(),
				})

				_, err = os.Stat(filepath.Join(tmp, "junit.xml"))
				Expect(err).NotTo(HaveOccurred())

				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.exitCode).To(Equal(123))
			})

			Context("command output passthrough", func() {
				It("passes through output of test command", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-functional-tests",
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
							"captain-cli-functional-test",
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
					suiteID := randomSuiteId()

					Expect(runCaptain(captainArgs{
						args: []string{
							"add", "quarantine",
							suiteID,
							"--file", "./x.rb",
							"--description", "is flaky",
						},
						env: getEnvWithoutAccessToken(),
					}).exitCode).To(Equal(0))

					// quarantine the failure

					result := runCaptain(captainArgs{
						args: []string{
							"run",
							suiteID,
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
							"captain-cli-functional-tests",
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
				// note these are tested as part of the add/remove quarantine tests
			})

			Context("with abq", func() {
				It("runs with ABQ_SET_EXIT_CODE=false when ABQ_SET_EXIT_CODE is unset", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
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
							"captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
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
							"captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
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
							"captain-cli-abq-test",
							"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
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

		Describe("captain [add|remove]", func() {
			actionBuilder := func(resource string, suiteID string) func(string) (captainResult, string) {
				read := func(path string) string {
					data, err := ioutil.ReadFile(path)
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}

				return func(action string) (captainResult, string) {
					result := runCaptain(captainArgs{
						args: []string{
							action, resource,
							suiteID,
							"--file", "./x.rb",
							"--description", "is flaky",
						},
						env: getEnvWithoutAccessToken(),
					})

					fileContents := read(fmt.Sprintf(".captain/%s/%ss.yaml", suiteID, resource))
					return result, fileContents
				}
			}

			Describe("quarantine", func() {
				It("can add and remove quarantines", func() {
					suiteID := randomSuiteId()

					runCaptainQuarantines := func() captainResult {
						return runCaptain(captainArgs{
							args: []string{
								"run",
								suiteID,
								"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
								"--retry-command", `echo "{{ tests }}"`,
								"-c", "bash -c 'exit 123'",
							},
							env: getEnvWithoutAccessToken(),
						})
					}
					By("before quarantining, tests should fail")

					Expect(runCaptainQuarantines().exitCode).To(Equal(123))

					action := actionBuilder("quarantine", suiteID)

					By("adding a quarantine, tests should succeed")
					addResult, quarantineFileContents := action("add")

					Expect(addResult.exitCode).To(Equal(0))
					Expect(quarantineFileContents).To(Equal("- file: ./x.rb\n" +
						"  description: is flaky\n"))

					Expect(runCaptainQuarantines().exitCode).To(Equal(0))

					By("running with non-quarantined failures, tests should fail")

					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"captain-cli-quarantine-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantined-with-other-errors.json",
							"-c", "bash -c 'exit 123'",
						},
						env: getEnvWithoutAccessToken(),
					})

					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
					Expect(result.exitCode).To(Equal(123))

					By("removing the quarantine, tests should fail again")

					removeResult, quarantineFileContents := action("remove")

					Expect(removeResult.exitCode).To(Equal(0))
					Expect(quarantineFileContents).To(Equal("[]\n"))

					Expect(runCaptainQuarantines().exitCode).To(Equal(123))
				})
			})

			Describe("flake", func() {
				It("can add and remove quarantines", func() {
					action := actionBuilder("flake", randomSuiteId())

					By("adding")
					result, fileContents := action("add")

					Expect(result.exitCode).To(Equal(0))
					Expect(fileContents).To(Equal("- file: ./x.rb\n" +
						"  description: is flaky\n"))

					By("removing")

					result, fileContents = action("remove")

					Expect(result.exitCode).To(Equal(0))
					Expect(fileContents).To(Equal("[]\n"))
				})
			})
		})
	})
})
