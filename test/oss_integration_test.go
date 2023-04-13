//go:build integration

package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/test/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(versionedPrefixForQuarantining()+"OSS mode Integration Tests", func() {
	randomSuiteId := func() string {
		// create a random suite ID so that concurrent tests don't try to read / write from the same timings files
		suiteUUID, err := uuid.NewRandom()
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		return suiteUUID.String()
	}
	prefix := "oss"

	Describe("captain --version", func() {
		It("renders the version I expect", func() {
			result := runCaptain(captainArgs{
				args: []string{"--version"},
				env:  make(map[string]string),
			})

			Expect(result.exitCode).To(Equal(0))
			withoutBackwardsCompatibility(func() {
				Expect(result.stdout).To(Equal(captain.Version))
				Expect(result.stderr).To(BeEmpty())
			})
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
						env: make(map[string]string),
					})

					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/x.rb,fixtures/integration-tests/partition/z.rb"))
					Expect(result.exitCode).To(Equal(0))

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
					})
				})

				It("sets partition 1 correctly when delimiter is set via env var", func() {
					env := make(map[string]string)
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

					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/x.rb,fixtures/integration-tests/partition/z.rb"))
					Expect(result.exitCode).To(Equal(0))

					withoutBackwardsCompatibility(func() {
						Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
					})
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
					env: make(map[string]string),
				})

				Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/x.rb fixtures/integration-tests/partition/z.rb"))
				Expect(result.exitCode).To(Equal(0))

				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
				})
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
					env: make(map[string]string),
				})

				Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/y.rb"))
				Expect(result.exitCode).To(Equal(0))

				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(ContainSubstring("No test file timings were matched."))
				})
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
					env: make(map[string]string),
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
					env: make(map[string]string),
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
					env: make(map[string]string),
				})

				Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb"))
				Expect(result.exitCode).To(Equal(0))
			})

			Context("with provider-provided node total & index", func() {
				var env map[string]string

				BeforeEach(func() {
					env = helpers.ReadEnvFromFile(".env.buildkite")
				})

				It("sets partition 2 correctly", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							suiteId,
							"fixtures/integration-tests/partition/*_spec.rb",
						},
						env: env,
					})

					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("prefers flag --index 0 to provider-sourced index 1", func() {
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							suiteId,
							"fixtures/integration-tests/partition/*_spec.rb",
							"--index", "0",
						},
						env: env,
					})

					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})

				It("prefers env var CAPTAIN_PARTITION_INDEX 0 to provider-sourced index 1", func() {
					env["CAPTAIN_PARTITION_INDEX"] = "0"

					env := helpers.ReadEnvFromFile(".env.buildkite")
					result := runCaptain(captainArgs{
						args: []string{
							"partition",
							suiteId,
							"fixtures/integration-tests/partition/*_spec.rb",
							"--index", "0",
						},
						env: env,
					})

					Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
					Expect(result.exitCode).To(Equal(0))
				})
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
					env: make(map[string]string),
				})

				Expect(result.stdout).To(Equal("fixtures/integration-tests/partition/a_spec.rb fixtures/integration-tests/partition/b_spec.rb fixtures/integration-tests/partition/c_spec.rb fixtures/integration-tests/partition/d_spec.rb"))
				Expect(result.exitCode).To(Equal(0))
			})
		})
	})

	Describe("captain run", func() {
		loadCaptainConfig := func(path string, substitution string) string {
			data, err := os.ReadFile(path)
			Expect(err).ToNot(HaveOccurred())
			return fmt.Sprintf(string(data), substitution)
		}
		It("fails if suite id is empty string", func() {
			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"--suite-id", "",
					"--test-results", "fixtures/integration-tests/rspec-passed.json",
					"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
				},
				env: make(map[string]string),
			})

			Expect(result.exitCode).ToNot(Equal(0))

			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: required flag \"suite-id\" not set"))
			})
		})

		It("fails if suite id is invalid", func() {
			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"--suite-id", "invalid-chars!./",
					"--test-results", "fixtures/integration-tests/rspec-passed.json",
					"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
				},
				env: make(map[string]string),
			})

			Expect(result.exitCode).ToNot(Equal(0))

			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("A suite ID can only contain alphanumeric characters, `_` and `-`."))
			})
		})

		It("fails & passes through exit code on failure when results files is missing", func() {
			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", "fixtures/integration-tests/does-not-exist.json",
					"-c", "bash -c 'exit 123'",
				},
				env: make(map[string]string),
			})
			Expect(result.exitCode).To(Equal(123))

			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				Expect(result.stdout).To(BeEmpty())
			})
		})

		It("fails & passes through exit code on failure", func() {
			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
					"-c", "bash -c 'exit 123'",
				},
				env: make(map[string]string),
			})

			Expect(result.exitCode).To(Equal(123))

			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
			})
		})

		It("allows using the -- command specification", func() {
			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined.json",
					"--", "bash", "-c", "exit 123",
				},
				env: make(map[string]string),
			})

			Expect(result.exitCode).To(Equal(123))
			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
			})
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
				env: make(map[string]string),
			})

			Expect(result.exitCode).To(Equal(123))
			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
			})
		})

		It("accepts --suite-id argument", func() {
			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"--suite-id", "captain-cli-functional-tests",
					"--test-results", "fixtures/integration-tests/rspec-passed.json",
					"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
				},
				env: make(map[string]string),
			})

			Expect(result.exitCode).To(Equal(0))
		})

		It("produces markdown-summary reports via config file", func() {
			tmp, err := os.MkdirTemp("", "*")
			Expect(err).NotTo(HaveOccurred())

			outputPath := filepath.Join(tmp, "markdown.md")
			os.Remove(outputPath)

			cfg := loadCaptainConfig("fixtures/integration-tests/captain-configs/markdown-summary-reporter.printf-yaml", outputPath)

			withCaptainConfig(cfg, tmp, func(configPath string) {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--config-file", configPath,
					},
					env: make(map[string]string),
				})

				_, err = os.Stat(outputPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.exitCode).To(Equal(123))

				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				})
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
				env: make(map[string]string),
			})

			_, err = os.Stat(filepath.Join(tmp, "markdown.md"))
			Expect(err).NotTo(HaveOccurred())

			Expect(result.exitCode).To(Equal(123))

			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
			})
		})

		It("produces rwx-v1-json reports via config file", func() {
			tmp, err := os.MkdirTemp("", "*")
			Expect(err).NotTo(HaveOccurred())

			outputPath := filepath.Join(tmp, "rwx-v1.json")
			os.Remove(outputPath)

			cfg := loadCaptainConfig("fixtures/integration-tests/captain-configs/rwx-v1-json-reporter.printf-yaml", outputPath)

			withCaptainConfig(cfg, tmp, func(configPath string) {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--config-file", configPath,
					},
					env: make(map[string]string),
				})

				_, err = os.Stat(outputPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.exitCode).To(Equal(123))

				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				})
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
				env: make(map[string]string),
			})

			_, err = os.Stat(filepath.Join(tmp, "rwx-v1.json"))
			Expect(err).NotTo(HaveOccurred())

			Expect(result.exitCode).To(Equal(123))

			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
			})
		})

		It("produces junit-xml reports via config file", func() {
			tmp, err := os.MkdirTemp("", "*")
			Expect(err).NotTo(HaveOccurred())

			outputPath := filepath.Join(tmp, "junit.xml")
			os.Remove(outputPath)

			cfg := loadCaptainConfig("fixtures/integration-tests/captain-configs/junit-xml-reporter.printf-yaml", outputPath)

			withCaptainConfig(cfg, tmp, func(configPath string) {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--config-file", configPath,
					},
					env: make(map[string]string),
				})

				_, err = os.Stat(outputPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(result.exitCode).To(Equal(123))

				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				})
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
				env: make(map[string]string),
			})

			_, err = os.Stat(filepath.Join(tmp, "junit.xml"))
			Expect(err).NotTo(HaveOccurred())

			Expect(result.exitCode).To(Equal(123))
			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
			})
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
					env: make(map[string]string),
				})

				Expect(result.stderr).To(ContainSubstring("def")) // prefix by a bunch of warnings about captain dir not existing
				Expect(result.stdout).To(ContainSubstring("abc\nghi"))
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
					env: make(map[string]string),
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
					env: make(map[string]string),
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
					env: make(map[string]string),
				})

				Expect(result.exitCode).To(Equal(0))
				Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'")) // indicative of a retry
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
					env: make(map[string]string),
				})

				Expect(result.exitCode).To(Equal(123))
				Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'")) // indicative of a retry
			})

			It("fails & passes through exit code on failure when configured via captain config", func() {
				_ = symlinkToNewPath("fixtures/integration-tests/rspec-failed-not-quarantined.json", fmt.Sprintf("with-config-%s", prefix))
				result := runCaptain(captainArgs{
					args: []string{
						"run", fmt.Sprintf("%s-quarantined-retries-with-config", prefix),
						"--config-file", "fixtures/integration-tests/retry-config.yaml",
					},
					env: make(map[string]string),
				})

				Expect(result.stdout).To(ContainSubstring("'./x.rb[1:1]'"))
				Expect(result.exitCode).To(Equal(123))
			})
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
					env: make(map[string]string),
				})

				Expect(result.exitCode).To(Equal(0))
				Expect(result.stdout).To(ContainSubstring("exit_code=false"))
			})

			It("runs with ABQ_SET_EXIT_CODE=false when ABQ_SET_EXIT_CODE is already set", func() {
				env := make(map[string]string)
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

				Expect(result.stdout).To(ContainSubstring("exit_code=false"))
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
					env: make(map[string]string),
				})

				Expect(result.stdout).To(ContainSubstring("state_file=/tmp/captain-abq-"))
				Expect(result.exitCode).To(Equal(0))
			})

			It("runs with previously set ABQ_STATE_FILE path when ABQ_STATE_FILE is set", func() {
				env := make(map[string]string)
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

				Expect(result.stdout).To(ContainSubstring("state_file=/tmp/functional-abq-1234.json"))
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
					env: make(map[string]string),
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
						env: make(map[string]string),
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
					env: make(map[string]string),
				})

				Expect(result.exitCode).To(Equal(123))
				withoutBackwardsCompatibility(func() {
					Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				})

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
