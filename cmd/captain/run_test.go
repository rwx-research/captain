package main_test

import (
	"strings"

	"github.com/spf13/cobra"

	main "github.com/rwx-research/captain-cli/cmd/captain"
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddFlags", func() {
	var (
		cmd     *cobra.Command
		cliArgs main.CliArgs
		err     error
		args    string
	)

	BeforeEach(func() {
		// set up the Cobra command and flags
		cmd = &cobra.Command{
			Use: "mycli",
			Run: func(cmd *cobra.Command, args []string) {}, // do nothing
		}
	})

	JustBeforeEach(func() {
		cliArgs = main.CliArgs{}
		main.AddFlags(cmd, &cliArgs)
		err = cmd.ParseFlags(strings.Split(args, " "))
	})

	Context("with Captain-specific environment variables", func() {
		BeforeEach(func() {
			setEnvFromFile("../../test/.env.captain")
		})

		It("parses the Captain options", func() {
			Expect(err).ToNot(HaveOccurred())

			provider := cliArgs.GenericProvider
			Expect(provider).To(Equal(providers.GenericEnv{
				Who:           "test",
				Branch:        "testing-env-vars",
				Sha:           "abc123",
				CommitMessage: "Testing env vars -- the commit message",
				BuildURL:      "https://jenkins.example.com/job/test/123/",
			}))
		})
	})
})
