package cli_test

import (
	"github.com/rwx-research/captain-cli/internal/cli"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RunConfig", func() {
	It("errs when retries are negative", func() {
		err := cli.RunConfig{Retries: -1}.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("retries must be >= 0"))
	})

	It("errs when retries are positive and the retry command is missing", func() {
		err := cli.RunConfig{Retries: 1, RetryCommandTemplate: ""}.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("retry-command must be provided if retries are > 0"))
	})

	It("is valid when retries are positive and the retry command is present", func() {
		err := cli.RunConfig{Retries: 1, RetryCommandTemplate: "some-command"}.Validate()
		Expect(err).NotTo(HaveOccurred())
	})

	It("is valid when retries are 0 and the retry command is missing", func() {
		err := cli.RunConfig{Retries: 0, RetryCommandTemplate: ""}.Validate()
		Expect(err).NotTo(HaveOccurred())
	})
})
