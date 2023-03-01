package cli_test

import (
	"github.com/rwx-research/captain-cli/internal/cli"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RunConfig", func() {
	Describe("Validate", func() {
		It("errs when max-tests-to-retry is neither an integer nor float percentage", func() {
			var err error

			err = cli.RunConfig{MaxTestsToRetry: "1.5"}.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("max-tests-to-retry must be either an integer or percentage"))

			err = cli.RunConfig{MaxTestsToRetry: "something"}.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("max-tests-to-retry must be either an integer or percentage"))

			err = cli.RunConfig{MaxTestsToRetry: "something%"}.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("max-tests-to-retry must be either an integer or percentage"))

			err = cli.RunConfig{MaxTestsToRetry: " 1 "}.Validate()
			Expect(err).NotTo(HaveOccurred())

			err = cli.RunConfig{MaxTestsToRetry: " 1.5% "}.Validate()
			Expect(err).NotTo(HaveOccurred())

			err = cli.RunConfig{MaxTestsToRetry: " 1% "}.Validate()
			Expect(err).NotTo(HaveOccurred())

			err = cli.RunConfig{MaxTestsToRetry: ""}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("errs when retries are positive and the retry command is missing", func() {
			err := cli.RunConfig{Retries: 1, RetryCommandTemplate: ""}.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("retry-command must be provided if retries or flaky-retries are > 0"))
		})

		It("errs when flaky-retries are positive and the retry command is missing", func() {
			err := cli.RunConfig{FlakyRetries: 1, RetryCommandTemplate: ""}.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("retry-command must be provided if retries or flaky-retries are > 0"))
		})

		It("is valid when retries are positive and the retry command is present", func() {
			err := cli.RunConfig{Retries: 1, RetryCommandTemplate: "some-command"}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are positive and the retry command is present", func() {
			err := cli.RunConfig{FlakyRetries: 1, RetryCommandTemplate: "some-command"}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are 0 and the retry command is missing", func() {
			err := cli.RunConfig{Retries: 0, RetryCommandTemplate: ""}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are 0 and the retry command is missing", func() {
			err := cli.RunConfig{FlakyRetries: 0, RetryCommandTemplate: ""}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are 0 and the retry command is present", func() {
			err := cli.RunConfig{Retries: 0, RetryCommandTemplate: "some-command"}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are 0 and the retry command is present", func() {
			err := cli.RunConfig{FlakyRetries: 0, RetryCommandTemplate: "some-command"}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are negative and the retry command is missing", func() {
			err := cli.RunConfig{Retries: -1, RetryCommandTemplate: ""}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are negative and the retry command is missing", func() {
			err := cli.RunConfig{FlakyRetries: -1, RetryCommandTemplate: ""}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when retries are negative and the retry command is present", func() {
			err := cli.RunConfig{Retries: -1, RetryCommandTemplate: "some-command"}.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid when flaky-retries are negative and the retry command is present", func() {
			err := cli.RunConfig{FlakyRetries: -1, RetryCommandTemplate: "some-command"}.Validate()
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
})
