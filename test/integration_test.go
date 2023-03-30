//go:build integration

package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/test/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type captainArgs struct {
	args []string
	env  map[string]string // note: we always set PATH and RWX_ACCESS_TOKEN
}

type captainResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func print(output string) {
	fmt.Fprintf(GinkgoWriter, output)
}

// used to debug tests
func printResults(result captainResult) {
	print(fmt.Sprintf("captain stdout: %s, captain stderr: %s, captain exit code: %d", result.stdout, result.stderr, result.exitCode))
}

func captainCmd(args captainArgs) *exec.Cmd {
	const captainPath = "../captain"
	_, err := os.Stat(captainPath)
	Expect(err).ToNot(HaveOccurred(), "integration tests depend on a captain binary in the root directory. This should be created automatically with `mage integrationTest` and `mage test`")

	cmd := exec.Command("../captain", args.args...)

	// always set the RWX_ACCESS_TOKEN and PATH
	env := []string{
		fmt.Sprintf("%s=%s", "PATH", os.Getenv("PATH")),
		fmt.Sprintf("%s=%s", "RWX_ACCESS_TOKEN", os.Getenv("RWX_ACCESS_TOKEN")),
	}

	for key, value := range args.env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Env = env

	fmt.Fprintf(GinkgoWriter, "Executing command: %s\n with env %s\n", cmd.String(), cmd.Env)

	return cmd
}

func runCaptain(args captainArgs) captainResult {
	cmd := captainCmd(args)
	var stdoutBuffer, stderrBuffer bytes.Buffer

	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err := cmd.Run()

	exitCode := 0

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		Expect(ok).To(BeTrue(), "captain exited with an error that wasn't an ExitError")
		exitCode = exitErr.ExitCode()
	}

	return captainResult{
		stdout:   strings.TrimSuffix(stdoutBuffer.String(), "\n"),
		stderr:   strings.TrimSuffix(stderrBuffer.String(), "\n"),
		exitCode: exitCode,
	}
}

func mergeMaps(into map[string]string, from map[string]string) map[string]string {
	for key, value := range from {
		into[key] = value
	}

	return into
}

func envAsMap() map[string]string {
	env := os.Environ()
	result := map[string]string{}

	for _, envVar := range env {
		parts := strings.SplitN(envVar, "=", 2)
		result[parts[0]] = parts[1]
	}

	return result
}

type envGenerator func() map[string]string

type sharedTestGen func(envGenerator, string)

var _ = Describe("Integration Tests", func() {
	withAndWithoutInheritedEnv := func(sharedTests sharedTestGen) {
		if os.Getenv("CI") != "" {
			// these tests are only run in CI
			Describe("using CI environment", func() {
				sharedTests(envAsMap, "inherited-env")
			})
		}

		if os.Getenv("ONLY_TEST_INHERITED_ENV") == "" {
			// don't run these on buildkite
			Describe("using the generic provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.captain")
				}, "generic")
			})

			Describe("using the GitHub Actions provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.github.actions")
				}, "github-actions")
			})

			Describe("using the GitLab CI provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.gitlab_ci")
				}, "gitlab-ci")
			})

			Describe("using the CircleCI provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.circleci")
				}, "circleci")
			})

			Describe("using the Buildkite provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.buildkite")
				}, "buildkite")
			})

		}
	}

	Context("With a valid RWX_ACCESS_TOKEN", func() {
		BeforeEach(func() {
			Expect(os.Getenv("RWX_ACCESS_TOKEN")).ToNot(BeEmpty(), "Integration tests require a valid RWX_ACCESS_TOKEN")
		})

		withAndWithoutInheritedEnv(func(getEnv envGenerator, prefix string) {
			Describe("captain run", func() {
				It("fails & passes through exit code on failure when results files is missing", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-functional-tests",
							"--test-results", "fixtures/integration-tests/does-not-exist.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'exit 123'",
						},
						env: getEnv(),
					})

					// Stderr being empty isn't ideal. See: https://github.com/rwx-research/captain-cli/issues/243
					Expect(result.stderr).To(Equal("Error: encountered error during execution of sub-process"))
					Expect(result.stdout).To(BeEmpty())
					Expect(result.exitCode).To(Equal(123))
				})

				It("fails & passes through exit code on failure", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-functional-tests",
							"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
							"--fail-on-upload-error",
							"-c", "bash -c 'exit 123'",
						},
						env: getEnv(),
					})

					Expect(result.stderr).To(Equal("Error: encountered error during execution of sub-process"))
					Expect(result.stdout).ToNot(BeEmpty())
					Expect(result.exitCode).To(Equal(123))
				})

				It("allows using the -- command specification", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-functional-tests",
							"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
							"--fail-on-upload-error",
							"--", "bash", "-c", "exit 123",
						},
						env: getEnv(),
					})

					Expect(result.stderr).To(Equal("Error: encountered error during execution of sub-process"))
					Expect(result.stdout).ToNot(BeEmpty())
					Expect(result.exitCode).To(Equal(123))
				})

				It("allows combining the --command specification with the -- specification", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"run",
							"--suite-id", "captain-cli-functional-tests",
							"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
							"--fail-on-upload-error",
							"-c", "bash",
							"--", "-c", "exit 123",
						},
						env: getEnv(),
					})

					Expect(result.stderr).To(Equal("Error: encountered error during execution of sub-process"))
					Expect(result.stdout).ToNot(BeEmpty())
					Expect(result.exitCode).To(Equal(123))
				})

				Context("quarantining", func() {
					It("succeeds when all failures quarantined", func() {
						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"--suite-id", "captain-cli-quarantine-test",
								"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
								"--fail-on-upload-error",
								"-c", "bash -c 'exit 2'",
							},
							env: getEnv(),
						})

						Expect(result.stderr).To(BeEmpty())
						Expect(result.stdout).ToNot(BeEmpty())
						Expect(result.exitCode).To(Equal(0))
					})

					It("fails & passes through exit code when not all failures quarantined", func() {
						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"--suite-id", "captain-cli-quarantine-test",
								"--test-results", "fixtures/integration-tests/rspec-quarantined-with-other-errors.json",
								"--fail-on-upload-error",
								"-c", "bash -c 'exit 123'",
							},
							env: getEnv(),
						})

						Expect(result.stderr).To(Equal("Error: encountered error during execution of sub-process"))
						Expect(result.stdout).ToNot(BeEmpty())
						Expect(result.exitCode).To(Equal(123))
					})
				})

				Context("command output passthrough", func() {
					It("passes through output of test command", func() {
						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"--suite-id", "captain-cli-quarantine-test",
								"--test-results", "fixtures/integration-tests/rspec-passed.json",
								"--fail-on-upload-error",
								"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
							},
							env: getEnv(),
						})

						Expect(result.stderr).To(Equal("def"))
						Expect(result.stdout).To(HavePrefix("abc\nghi\n"))
						Expect(result.exitCode).To(Equal(0))
					})

					It("passes through output of test command in the correct order", func() {
						cmd := captainCmd(captainArgs{
							args: []string{
								"run",
								"--suite-id", "captain-cli-quarantine-test",
								"--test-results", "fixtures/integration-tests/rspec-passed.json",
								"--fail-on-upload-error",
								"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
							},
							env: getEnv(),
						})

						combinedOutputBytes, err := cmd.CombinedOutput()
						Expect(err).ToNot(HaveOccurred())

						Expect(string(combinedOutputBytes)).To(HavePrefix("abc\ndef\nghi\n"))
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
								"--fail-on-upload-error",
								"--retries", "1",
								"--retry-command", `echo "{{ tests }}"`,
								"-c", "bash -c 'exit 123'",
							},
							env: getEnv(),
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
								"--fail-on-upload-error",
								"--retries", "1",
								"--retry-command", `echo "{{ tests }}"`,
								"-c", "bash -c 'exit 123'",
							},
							env: getEnv(),
						})

						Expect(result.stderr).To(Equal("Error: encountered error during execution of sub-process"))
						Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
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
							env: getEnv(),
						})

						Expect(result.stderr).To(BeEmpty())
						Expect(result.stdout).To(HavePrefix("exit_code=false"))
						Expect(result.exitCode).To(Equal(0))
					})

					It("runs with ABQ_SET_EXIT_CODE=false when ABQ_SET_EXIT_CODE is already set", func() {
						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"--suite-id", "captain-cli-abq-test",
								"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
								"--fail-on-upload-error",
								"-c", "bash -c 'echo exit_code=$ABQ_SET_EXIT_CODE'",
							},
							env: mergeMaps(getEnv(), map[string]string{"ABQ_SET_EXIT_CODE": "1234"}),
						})

						Expect(result.stderr).To(BeEmpty())
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
							env: getEnv(),
						})

						Expect(result.stderr).To(BeEmpty())
						Expect(result.stdout).To(HavePrefix("state_file=/tmp/captain-abq-"))
						Expect(result.exitCode).To(Equal(0))
					})

					It("runs with previously set ABQ_STATE_FILE path when ABQ_STATE_FILE is set", func() {
						result := runCaptain(captainArgs{
							args: []string{
								"run",
								"--suite-id", "captain-cli-abq-test",
								"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
								"--fail-on-upload-error",
								"-c", "bash -c 'echo state_file=$ABQ_STATE_FILE'",
							},
							env: mergeMaps(getEnv(), map[string]string{"ABQ_STATE_FILE": "/tmp/functional-abq-1234.json"}),
						})

						Expect(result.stderr).To(BeEmpty())
						Expect(result.stdout).To(HavePrefix("state_file=/tmp/functional-abq-1234.json"))
						Expect(result.exitCode).To(Equal(0))
					})
				})
			})

			Describe("captain quarantine", func() {
				It("succeeds when all failures quarantined", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"quarantine",
							"--suite-id", "captain-cli-quarantine-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantine.json",
							"-c", "bash -c 'exit 2'",
						},
						env: getEnv(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).ToNot(BeEmpty())
					Expect(result.exitCode).To(Equal(0))
				})

				It("fails & passes through exit code when not all failures quarantined", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"quarantine",
							"--suite-id", "captain-cli-quarantine-test",
							"--test-results", "fixtures/integration-tests/rspec-quarantined-with-other-errors.json",
							"-c", "bash -c 'exit 123'",
						},
						env: getEnv(),
					})

					Expect(result.stderr).To(Equal("Error: encountered error during execution of sub-process"))
					Expect(result.stdout).ToNot(BeEmpty())
					Expect(result.exitCode).To(Equal(123))
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
							env: getEnv(),
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
							env: getEnv(),
						})

						Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
						Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/y.rb"))
						Expect(result.exitCode).To(Equal(0))
					})
				})

				Context("with timings", func() {
					// to regenerate timings, edit rspec-partition.json and then run
					// 1. captain upload results test/fixtures/integration-tests/partition/rspec-partition.json --suite-id captain-cli-functional-tests
					// 2. change the CAPTAIN_SHA parameter to pull in the new timings

					It("sets partition 1 correctly", func() {
						result := runCaptain(captainArgs{
							args: []string{
								"partition",
								"fixtures/integration-tests/partition/*_spec.rb",
								"--suite-id", "captain-cli-functional-tests",
								"--index", "0",
								"--total", "2",
							},
							env: getEnv(),
						})

						Expect(result.stderr).To(BeEmpty())
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
							env: getEnv(),
						})

						Expect(result.stderr).To(BeEmpty())
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
							env: getEnv(),
						})

						Expect(result.stderr).To(BeEmpty())
						Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
						Expect(result.exitCode).To(Equal(0))
					})
				})
			})

			Describe("captain upload", func() {
				It("short circuits when there's nothing to upload", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"upload", "results",
							"nonexistingfile.json",
							"--suite-id", "captain-cli-functional-tests",
						},
						env: getEnv(),
					})

					Expect(result.stderr).To(BeEmpty())
					Expect(result.stdout).To(BeEmpty())
					Expect(result.exitCode).To(Equal(0))
				})
			})
		})
	})

	Context("Without a valid RWX_ACCESS_TOKEN", func() {
		withAndWithoutInheritedEnv(func(getEnv envGenerator, _prefix string) {
			Describe("captain --version", func() {
				It("renders the version I expect", func() {
					result := runCaptain(captainArgs{
						args: []string{"--version"},
						env:  getEnv(),
					})

					Expect(result.exitCode).To(Equal(0))
					Expect(result.stdout).To(Equal(captain.Version))
					Expect(result.stdout).To(MatchRegexp(`^v\d+\.\d+\.\d+$`))
					Expect(result.stderr).To(BeEmpty())
				})
			})
		})
	})
})
