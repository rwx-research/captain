package main_test

import (
	"strings"

	"github.com/spf13/cobra"

	main "github.com/rwx-research/captain-cli/cmd/captain"
	"github.com/rwx-research/captain-cli/internal/providers"
	"github.com/rwx-research/captain-cli/test/helpers"

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
		err = main.AddFlags(cmd, &cliArgs)
		Expect(err).ToNot(HaveOccurred())
		err = cmd.ParseFlags(strings.Split(args, " "))
	})

	Context("with Captain-specific environment variables", func() {
		BeforeEach(func() {
			helpers.UnsetCIEnv()
			helpers.SetEnvFromFile("../../test/.env.captain")
		})

		It("parses the Captain options", func() {
			Expect(err).ToNot(HaveOccurred())

			provider := cliArgs.GenericProvider
			Expect(provider).To(Equal(providers.GenericEnv{
				Who:           "test",
				Branch:        "testing-env-vars",
				Sha:           "1fc108cab0bb46083c6cdd50f8cd1deb5005e235",
				CommitMessage: "Testing env vars -- the commit message",
				BuildURL:      "https://jenkins.example.com/job/test/123/",
			}))
		})
	})
})
