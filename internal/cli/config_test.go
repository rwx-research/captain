package cli_test

import (
	"strings"

	"go.uber.org/zap/zaptest"
	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RunConfig", func() {
	logger := zaptest.NewLogger(GinkgoT()).Sugar()

	Describe("Validate", func() {
		It("errs when max-tests-to-retry is neither an integer nor float percentage", func() {
			var err error

			err = cli.RunConfig{MaxTestsToRetry: "1.5"}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unsupported --max-tests-to-retry value"))

			err = cli.RunConfig{MaxTestsToRetry: "something"}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unsupported --max-tests-to-retry value"))

			err = cli.RunConfig{MaxTestsToRetry: "something%"}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unsupported --max-tests-to-retry value"))

			err = cli.RunConfig{MaxTestsToRetry: " 1 "}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())

			err = cli.RunConfig{MaxTestsToRetry: " 1.5% "}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())

			err = cli.RunConfig{MaxTestsToRetry: " 1% "}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())

			err = cli.RunConfig{MaxTestsToRetry: ""}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("errs when retries are positive and the retry command is missing", func() {
			err := cli.RunConfig{Retries: 1, RetryCommandTemplate: ""}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing retry command"))
		})

		It("errs when flaky-retries are positive and the retry command is missing", func() {
			err := cli.RunConfig{FlakyRetries: 1, RetryCommandTemplate: ""}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing retry command"))
		})

		It("is valid when retries are positive and the retry command is present", func() {
			err := cli.RunConfig{Retries: 1, RetryCommandTemplate: "some-command"}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are positive and the retry command is present", func() {
			err := cli.RunConfig{FlakyRetries: 1, RetryCommandTemplate: "some-command"}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are 0 and the retry command is missing", func() {
			err := cli.RunConfig{Retries: 0, RetryCommandTemplate: ""}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are 0 and the retry command is missing", func() {
			err := cli.RunConfig{FlakyRetries: 0, RetryCommandTemplate: ""}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are 0 and the retry command is present", func() {
			err := cli.RunConfig{Retries: 0, RetryCommandTemplate: "some-command"}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are 0 and the retry command is present", func() {
			err := cli.RunConfig{FlakyRetries: 0, RetryCommandTemplate: "some-command"}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are negative and the retry command is missing", func() {
			err := cli.RunConfig{Retries: -1, RetryCommandTemplate: ""}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are negative and the retry command is missing", func() {
			err := cli.RunConfig{FlakyRetries: -1, RetryCommandTemplate: ""}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are negative and the retry command is present", func() {
			err := cli.RunConfig{Retries: -1, RetryCommandTemplate: "some-command"}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are negative and the retry command is present", func() {
			err := cli.RunConfig{FlakyRetries: -1, RetryCommandTemplate: "some-command"}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are disabled and nothing else retry-related is configured", func() {
			err := cli.RunConfig{RetriesDisabled: true, Retries: -1, FlakyRetries: -1}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("errs when retries are disabled and a retry command is configured", func() {
			err := cli.RunConfig{RetriesDisabled: true, RetryCommandTemplate: "some-command"}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Retries are disabled"))
		})

		It("errs when retries are disabled and retries are positive", func() {
			err := cli.RunConfig{RetriesDisabled: true, Retries: 1}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Retries are disabled"))
		})

		It("errs when retries are disabled and flaky-retries are positive", func() {
			err := cli.RunConfig{RetriesDisabled: true, FlakyRetries: 1}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Retries are disabled"))
		})

		It("errs when additional artifact paths are provided without intermediate artifacts path", func() {
			err := cli.RunConfig{
				AdditionalArtifactPaths:   []string{"coverage/**/*"},
				IntermediateArtifactsPath: "",
			}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing intermediate artifacts path"))
		})

		It("is valid when additional artifact paths are provided with intermediate artifacts path", func() {
			err := cli.RunConfig{
				AdditionalArtifactPaths:   []string{"coverage/**/*"},
				IntermediateArtifactsPath: "/tmp/artifacts",
			}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when no additional artifact paths are provided", func() {
			err := cli.RunConfig{
				AdditionalArtifactPaths:   []string{},
				IntermediateArtifactsPath: "",
			}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("errs when partitioning and partition config is missing suite id", func() {
			err := cli.RunConfig{
				PartitionCommandTemplate: "something {{ testFiles }}",
				PartitionConfig: cli.PartitionConfig{
					PartitionNodes: config.PartitionNodes{
						Index: 0,
						Total: 1,
					},
				},
			}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing suite ID"))
		})

		It("errs when partitioning and partition config is missing test file paths", func() {
			err := cli.RunConfig{
				PartitionCommandTemplate: "something {{ testFiles }}",
				PartitionConfig: cli.PartitionConfig{
					SuiteID: "your-suite",
					PartitionNodes: config.PartitionNodes{
						Index: 0,
						Total: 1,
					},
				},
			}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing test file paths"))
		})

		It("errs when partition config has out of bound indices", func() {
			err := cli.RunConfig{
				PartitionCommandTemplate: "something {{ testFiles }}",
				PartitionConfig: cli.PartitionConfig{
					SuiteID: "your-suite",
					PartitionNodes: config.PartitionNodes{
						Index: 2,
						Total: 1,
					},
				},
			}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unsupported partitioning setup"))
		})

		It("errs when partitioning and partition config has nonsense index", func() {
			err := cli.RunConfig{
				PartitionCommandTemplate: "something {{ testFiles }}",
				PartitionConfig: cli.PartitionConfig{
					SuiteID: "your-suite",
					PartitionNodes: config.PartitionNodes{
						Index: -1,
						Total: 1,
					},
				},
			}.Validate(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing partition index"))
		})

		It("is not an error when partitioning is configured but not currently partitioning", func() {
			err := cli.RunConfig{
				PartitionCommandTemplate: "something {{ testFiles }}",
				PartitionConfig: cli.PartitionConfig{
					SuiteID: "your-suite",
					PartitionNodes: config.PartitionNodes{
						Index: -1,
						Total: -1,
					},
				},
			}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when partitioning and partition config has command and globs", func() {
			err := cli.RunConfig{
				PartitionCommandTemplate: "something {{ testFiles }}",
				PartitionConfig: cli.PartitionConfig{
					SuiteID: "your-suite",
					PartitionNodes: config.PartitionNodes{
						Index: 1,
						Total: 2,
					},
					TestFilePaths: []string{"spec/**/*_spec.rb"},
				},
			}.Validate(logger)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("MaxTestsToRetryCount", func() {
		It("returns the count when set", func() {
			count, err := cli.RunConfig{MaxTestsToRetry: "1"}.MaxTestsToRetryCount()
			Expect(err).NotTo(HaveOccurred())
			Expect(*count).To(Equal(1))
		})

		It("returns nil when not set", func() {
			count, err := cli.RunConfig{MaxTestsToRetry: ""}.MaxTestsToRetryCount()
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(BeNil())
		})

		It("returns nil when a percentage", func() {
			count, err := cli.RunConfig{MaxTestsToRetry: "1.5%"}.MaxTestsToRetryCount()
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(BeNil())
		})
	})

	Describe("MaxTestsToRetryPercentage", func() {
		It("returns the percentage when set", func() {
			percentage, err := cli.RunConfig{MaxTestsToRetry: "1.5%"}.MaxTestsToRetryPercentage()
			Expect(err).NotTo(HaveOccurred())
			Expect(*percentage).To(Equal(1.5))
		})

		It("returns nil when not set", func() {
			percentage, err := cli.RunConfig{MaxTestsToRetry: ""}.MaxTestsToRetryPercentage()
			Expect(err).NotTo(HaveOccurred())
			Expect(percentage).To(BeNil())
		})

		It("returns nil when a count", func() {
			percentage, err := cli.RunConfig{MaxTestsToRetry: "1"}.MaxTestsToRetryPercentage()
			Expect(err).NotTo(HaveOccurred())
			Expect(percentage).To(BeNil())
		})
	})

	Describe("IsRunningPartition", func() {
		It("returns false when partition command template is not set", func() {
			Expect(cli.RunConfig{}.IsRunningPartition()).To(Equal(false))
		})

		It("returns false when partition command template is set, but partition nodes are defaulted to -1", func() {
			rc := cli.RunConfig{
				PartitionCommandTemplate: "bin/rspec {{testFiles}}",
				PartitionConfig: cli.PartitionConfig{
					PartitionNodes: config.PartitionNodes{
						Index: -1,
						Total: -1,
					},
				},
			}
			Expect(rc.IsRunningPartition()).To(Equal(false))
		})

		It("returns false when partition command template is set, but neither partition nodes are unset", func() {
			rc := cli.RunConfig{
				PartitionCommandTemplate: "bin/rspec {{testFiles}}",
				PartitionConfig:          cli.PartitionConfig{},
			}
			Expect(rc.IsRunningPartition()).To(Equal(false))
		})

		It("returns true when partition command template and partition total is set", func() {
			rc := cli.RunConfig{
				PartitionCommandTemplate: "bin/rspec {{testFiles}}",
				PartitionConfig: cli.PartitionConfig{
					PartitionNodes: config.PartitionNodes{
						Index: 0,
						Total: 2,
					},
				},
			}
			Expect(rc.IsRunningPartition()).To(Equal(true))
		})
	})
})

var _ = Describe("SuiteConfigRetries", func() {
	decode := func(document string) (cli.ConfigFile, error) {
		var configFile cli.ConfigFile
		decoder := yaml.NewDecoder(strings.NewReader(document))
		decoder.KnownFields(true)
		err := decoder.Decode(&configFile)
		return configFile, err
	}

	It("parses `retries: disabled`", func() {
		configFile, err := decode("test-suites:\n  some-suite:\n    retries: disabled\n")
		Expect(err).NotTo(HaveOccurred())
		Expect(configFile.TestSuites["some-suite"].Retries.Disabled).To(BeTrue())
	})

	It("still parses a retries mapping", func() {
		configFile, err := decode(
			"test-suites:\n  some-suite:\n    retries:\n      attempts: 2\n      command: some-command\n",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(configFile.TestSuites["some-suite"].Retries.Disabled).To(BeFalse())
		Expect(configFile.TestSuites["some-suite"].Retries.Attempts).To(Equal(2))
		Expect(configFile.TestSuites["some-suite"].Retries.Command).To(Equal("some-command"))
	})

	It("still rejects unknown fields in a retries mapping", func() {
		_, err := decode("test-suites:\n  some-suite:\n    retries:\n      not-a-field: true\n")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not-a-field"))
	})

	It("rejects other strings", func() {
		_, err := decode("test-suites:\n  some-suite:\n    retries: sometimes\n")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`expected "disabled" or a mapping`))
	})
})
