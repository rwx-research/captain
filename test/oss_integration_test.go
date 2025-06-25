//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/google/uuid"
	"github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/test/helpers"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

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
			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passed.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"--suite-id", "",
					"--test-results", testResultsPath,
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
			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passed.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"--suite-id", "invalid-chars!./",
					"--test-results", testResultsPath,
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
			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/does-not-exist.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", testResultsPath,
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
			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", testResultsPath,
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
			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", testResultsPath,
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
			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", testResultsPath,
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
			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passed.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"--suite-id", "captain-cli-functional-tests",
					"--test-results", testResultsPath,
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

			_, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

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

			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", testResultsPath,
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

			_, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

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

			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", testResultsPath,
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

			_, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

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

			testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", testResultsPath,
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

		It("accepts the language and framework CLI flags and parses with their parser", func() {
			tmp, err := os.MkdirTemp("", "*")
			Expect(err).NotTo(HaveOccurred())

			os.Remove(filepath.Join(tmp, "rwx-v1.json"))

			copyFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", "fixtures/integration-tests/rspec-failed-not-quarantined-for-snapshot-29834.json")
			cleanup := func() {
				os.Remove("fixtures/integration-tests/rspec-failed-not-quarantined-for-snapshot-29834.json")
			}
			defer cleanup()

			result := runCaptain(captainArgs{
				args: []string{
					"run",
					"captain-cli-functional-tests",
					"--test-results", "fixtures/integration-tests/rspec-failed-not-quarantined-for-snapshot-29834.json",
					"--language", "Ruby",
					"--framework", "RSpec",
					"--reporter", fmt.Sprintf("rwx-v1-json=%v", filepath.Join(tmp, "rwx-v1.json")),
					"-c", "bash -c 'exit 123'",
				},
				env: make(map[string]string),
			})

			data, err := ioutil.ReadFile(filepath.Join(tmp, "rwx-v1.json"))
			Expect(err).ToNot(HaveOccurred())

			Expect(result.exitCode).To(Equal(123))
			withoutBackwardsCompatibility(func() {
				Expect(result.stderr).To(ContainSubstring("Error: test suite exited with non-zero exit code"))
				cupaloy.SnapshotT(GinkgoT(), data)
			})
		})

		Context("command output passthrough", func() {
			It("passes through output of test command", func() {
				testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passed.json", prefix)
				defer cleanup()

				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--test-results", testResultsPath,
						"-c", "bash -c 'echo abc; echo def 1>&2; echo ghi'",
					},
					env: make(map[string]string),
				})

				Expect(result.stderr).To(ContainSubstring("def")) // prefix by a bunch of warnings about captain dir not existing
				Expect(result.stdout).To(ContainSubstring("abc\nghi"))
				Expect(result.exitCode).To(Equal(0))
			})

			It("passes through output of test command in the correct order", func() {
				testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-passed.json", prefix)
				defer cleanup()

				cmd := captainCmd(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-test",
						"--test-results", testResultsPath,
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
				testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-quarantine.json", prefix)
				defer cleanup()

				result := runCaptain(captainArgs{
					args: []string{
						"run",
						suiteID,
						"--test-results", testResultsPath,
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
				testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", prefix)
				defer cleanup()

				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-functional-tests",
						"--test-results", testResultsPath,
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
				_, cleanup := createUniqueFile("fixtures/integration-tests/rspec-failed-not-quarantined.json", fmt.Sprintf("with-config-%s", prefix))
				defer cleanup()
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

		Context("when partitioning is configured but flags aren't provided", func() {
			It("runs the default run command", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
					},
					env: make(map[string]string),
				})

				Expect(result.exitCode).To(Equal(123))
				Expect(result.stdout).To(Equal("default run"))
			})
		})
	})

	Describe("captain run w/ partition", func() {
		Context("given minimal partition config", func() {
			It("runs the partition command against the provided globs", func() {
				partitionResult0 := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
						"--partition-index", "0",
						"--partition-total", "2",
					},
					env: make(map[string]string),
				})

				Expect(partitionResult0.exitCode).To(Equal(0))
				Expect(partitionResult0.stdout).To(ContainSubstring("test 'fixtures/integration-tests/partition/a_spec.rb' 'fixtures/integration-tests/partition/c_spec.rb' 'fixtures/integration-tests/partition/x.rb' 'fixtures/integration-tests/partition/z.rb'"))

				partitionResult1 := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
						"--partition-index", "1",
						"--partition-total", "2",
					},
					env: make(map[string]string),
				})

				Expect(partitionResult1.exitCode).To(Equal(0))
				Expect(partitionResult1.stdout).To(ContainSubstring("test 'fixtures/integration-tests/partition/b_spec.rb' 'fixtures/integration-tests/partition/d_spec.rb' 'fixtures/integration-tests/partition/y.rb'"))
			})

			It("accepts env vars instead of command line args", func() {
				env := make(map[string]string)
				env["CAPTAIN_PARTITION_INDEX"] = "0"
				env["CAPTAIN_PARTITION_TOTAL"] = "2"
				partitionResult0 := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
					},
					env: env,
				})

				Expect(partitionResult0.exitCode).To(Equal(0))
				Expect(partitionResult0.stdout).To(ContainSubstring("test 'fixtures/integration-tests/partition/a_spec.rb' 'fixtures/integration-tests/partition/c_spec.rb' 'fixtures/integration-tests/partition/x.rb' 'fixtures/integration-tests/partition/z.rb'"))
			})
		})

		Context("given partition with multiple globs", func() {
			It("partitions the files using all glob patterns", func() {
				partitionResult0 := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition-with-multiple-globs",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
						"--partition-index", "0",
						"--partition-total", "2",
					},
					env: make(map[string]string),
				})

				Expect(partitionResult0.exitCode).To(Equal(0))
				Expect(partitionResult0.stdout).To(ContainSubstring("test 'fixtures/integration-tests/partition/a_spec.rb' 'fixtures/integration-tests/partition/c_spec.rb' 'fixtures/integration-tests/partition/x.rb' 'fixtures/integration-tests/partition/z.rb'"))

				partitionResult1 := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition-with-multiple-globs",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
						"--partition-index", "1",
						"--partition-total", "2",
					},
					env: make(map[string]string),
				})

				Expect(partitionResult1.exitCode).To(Equal(0))
				Expect(partitionResult1.stdout).To(ContainSubstring("test 'fixtures/integration-tests/partition/b_spec.rb' 'fixtures/integration-tests/partition/d_spec.rb' 'fixtures/integration-tests/partition/y.rb'"))
			})
		})

		Context("given partition with custom delimiter globs", func() {
			It("formats the output using the delimiter", func() {
				partitionResult0 := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition-with-custom-delimiter",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
						"--partition-index", "0",
						"--partition-total", "2",
					},
					env: make(map[string]string),
				})

				Expect(partitionResult0.exitCode).To(Equal(0))
				Expect(partitionResult0.stdout).To(ContainSubstring("test 'fixtures/integration-tests/partition/a_spec.rb' -hi- 'fixtures/integration-tests/partition/c_spec.rb' -hi- 'fixtures/integration-tests/partition/x.rb' -hi- 'fixtures/integration-tests/partition/z.rb'"))
			})
		})

		Context("when partition configured but partition flags are not provided", func() {
			It("uses the default run command", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
					},
					env: make(map[string]string),
				})

				Expect(result.exitCode).To(Equal(123))
				Expect(result.stdout).To(ContainSubstring("default run"))
			})
		})

		Context("when partitioning results in an empty partition", func() {
			It("noops the command and prints an error to the screen", func() {
				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"oss-run-with-partition",
						"--config-file", "fixtures/integration-tests/partition-config.yaml",
						"--partition-index", "29",
						"--partition-total", "30",
					},
					env: make(map[string]string),
				})

				Expect(result.exitCode).To(Equal(0))
				Expect(result.stderr).To(ContainSubstring("Partition 29/30 contained no test files. 7/30 partitions were utilized. We recommend you set --partition-total no more than 7"))
			})
		})

		It("accepts env vars instead of command line args", func() {
			env := make(map[string]string)
			env["CAPTAIN_PARTITION_INDEX"] = "0"
			env["CAPTAIN_PARTITION_TOTAL"] = "2"
			partitionResult0 := runCaptain(captainArgs{
				args: []string{
					"run",
					"oss-run-with-partition",
					"--config-file", "fixtures/integration-tests/partition-config.yaml",
				},
				env: env,
			})

			Expect(partitionResult0.exitCode).To(Equal(0))
			Expect(partitionResult0.stdout).To(ContainSubstring("test 'fixtures/integration-tests/partition/a_spec.rb' 'fixtures/integration-tests/partition/c_spec.rb' 'fixtures/integration-tests/partition/x.rb' 'fixtures/integration-tests/partition/z.rb'"))
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
				testResultsPath, cleanup := createUniqueFile("fixtures/integration-tests/rspec-quarantine.json", prefix)
				defer cleanup()

				runCaptainQuarantines := func() captainResult {
					return runCaptain(captainArgs{
						args: []string{
							"run",
							suiteID,
							"--test-results", testResultsPath,
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

				// Restore the file which was deleted by the first run
				copyFile("fixtures/integration-tests/rspec-quarantine.json", testResultsPath)

				Expect(runCaptainQuarantines().exitCode).To(Equal(0))

				By("running with non-quarantined failures, tests should fail")

				testResultsPath2, cleanup2 := createUniqueFile("fixtures/integration-tests/rspec-quarantined-with-other-errors.json", prefix)
				defer cleanup2()

				result := runCaptain(captainArgs{
					args: []string{
						"run",
						"captain-cli-quarantine-test",
						"--test-results", testResultsPath2,
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

	Describe("captain merge", func() {
		It("parses and merges provided results", func() {
			result := runCaptain(captainArgs{
				args: []string{"merge", "fixtures/merge/rspec-one.json", "fixtures/merge/rspec-two.json", "--print-summary"},
				env:  make(map[string]string),
			})

			Expect(result.exitCode).To(Equal(0))
			withoutBackwardsCompatibility(func() {
				cupaloy.SnapshotT(GinkgoT(), result.stdout)
			})
		})

		It("reports to specified paths", func() {
			tmp, err := os.MkdirTemp("", "*")
			Expect(err).NotTo(HaveOccurred())

			os.Remove(filepath.Join(tmp, "rwx-v1.json"))

			result := runCaptain(captainArgs{
				args: []string{"merge", "fixtures/merge/rspec-one.json", "fixtures/merge/rspec-two.json", "--reporter", fmt.Sprintf("rwx-v1-json=%v", filepath.Join(tmp, "rwx-v1.json"))},
				env:  make(map[string]string),
			})

			Expect(result.exitCode).To(Equal(0))

			data, err := ioutil.ReadFile(filepath.Join(tmp, "rwx-v1.json"))
			Expect(err).ToNot(HaveOccurred())

			withoutBackwardsCompatibility(func() {
				cupaloy.SnapshotT(GinkgoT(), data)
			})
		})

		It("supports globs", func() {
			result := runCaptain(captainArgs{
				args: []string{"merge", "fixtures/merge/rspec-*.json", "--print-summary"},
				env:  make(map[string]string),
			})

			Expect(result.exitCode).To(Equal(0))
			withoutBackwardsCompatibility(func() {
				cupaloy.SnapshotT(GinkgoT(), result.stdout)
			})
		})
	})

	withoutBackwardsCompatibility(func() {
		// `captain parse results` is a hidden command and not part of our public interface, so we don't
		// assure backwards compatibility
		Describe("captain parse", func() {
			Describe("results", func() {
				It("produces RWX JSON with the parsers", func() {
					result := runCaptain(captainArgs{
						args: []string{"parse", "results", "fixtures/rspec.json"},
						env:  make(map[string]string),
					})

					Expect(result.exitCode).To(Equal(0))
					cupaloy.SnapshotT(GinkgoT(), result.stdout)
				})

				It("does not add the parsed file to derivedFrom when it's already in the RWX format", func() {
					result := runCaptain(captainArgs{
						args: []string{"parse", "results", "fixtures/rwx/v1_not_derived.json"},
						env:  make(map[string]string),
					})

					Expect(result.exitCode).To(Equal(0))

					var testResults v1.TestResults

					err := json.NewDecoder(strings.NewReader(result.stdout)).Decode(&testResults)
					Expect(err).NotTo(HaveOccurred())
					Expect(testResults.DerivedFrom).To(BeNil())
				})

				It("leaves the derived from contents exactly as it was when it's already in the RWX format", func() {
					result := runCaptain(captainArgs{
						args: []string{"parse", "results", "fixtures/rwx/v1.json"},
						env:  make(map[string]string),
					})

					Expect(result.exitCode).To(Equal(0))

					var testResults v1.TestResults

					err := json.NewDecoder(strings.NewReader(result.stdout)).Decode(&testResults)
					Expect(err).NotTo(HaveOccurred())
					Expect(testResults.DerivedFrom[0].Contents).To(Equal("base64encodedoriginalfile"))
				})

				It("supports globs", func() {
					result := runCaptain(captainArgs{
						args: []string{"parse", "results", "fixtures/rs*ec.json"},
						env:  make(map[string]string),
					})

					Expect(result.exitCode).To(Equal(0))
					cupaloy.SnapshotT(GinkgoT(), result.stdout)
				})
			})
		})
	})
})
