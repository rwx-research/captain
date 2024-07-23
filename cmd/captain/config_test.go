package main_test

import (
	"os"

	"github.com/spf13/cobra"

	captain "github.com/rwx-research/captain-cli/cmd/captain"
	main "github.com/rwx-research/captain-cli/cmd/captain"
	"github.com/rwx-research/captain-cli/test/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("InitConfig", func() {
	var cmd *cobra.Command
	var cliArgs main.CliArgs

	BeforeEach(func() {
		cliArgs = main.CliArgs{}
		cmd = &cobra.Command{
			Use:                "mycli",
			Run:                func(_ *cobra.Command, _ []string) {}, // do nothing
			DisableFlagParsing: true,                                  // without this we can't call Execute()
		}
		Expect(cmd.Execute()).ToNot(HaveOccurred()) // before Execute, cmd.Context() is nil
		helpers.UnsetCIEnv()
		os.Setenv("CAPTAIN_SUITE_ID", "some-suite")
		err := captain.ConfigureRootCmd(cmd, &cliArgs)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("no environment variables", func() {
		It("uses cobra's default values", func() {
			err := captain.AddFlags(cmd, &cliArgs)
			Expect(err).ToNot(HaveOccurred())
			err = cmd.ParseFlags([]string{})
			Expect(err).ToNot(HaveOccurred())

			Expect(cliArgs.Retries).To(Equal(-1))

			cfg, err := captain.InitConfig(cmd, cliArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.TestSuites).To(HaveKey("some-suite"))
			Expect(cfg.TestSuites["some-suite"].Retries.Attempts).To(Equal(-1))
			Expect(cfg.TestSuites["some-suite"].Retries.FlakyAttempts).To(Equal(-1))
		})
	})

	Context("with GitHub Actions environment variables", func() {
		BeforeEach(func() {
			helpers.SetEnvFromFile("../../test/.env.github.actions")
		})

		It("parses the GitHub options", func() {
			cfg, err := captain.InitConfig(cmd, cliArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.ProvidersEnv.GitHub.Detected).To(BeTrue())
			Expect(cfg.ProvidersEnv.GitHub.Repository).To(Equal("rwx-research/captain-cli"))
			Expect(cfg.ProvidersEnv.GitHub.Name).To(Equal("some-job-name"))
			Expect(cfg.ProvidersEnv.GitHub.Attempt).To(Equal("4"))
			Expect(cfg.ProvidersEnv.GitHub.ID).To(Equal("1234"))
			Expect(cfg.ProvidersEnv.GitHub.CommitSha).To(Equal("1fc108cab0bb46083c6cdd50f8cd1deb5005e235"))
			Expect(cfg.ProvidersEnv.GitHub.ExecutingActor).To(Equal("captain-cli-test"))
			Expect(cfg.ProvidersEnv.GitHub.TriggeringActor).To(Equal("captain-cli-test-trigger"))
			Expect(cfg.ProvidersEnv.GitHub.EventName).To(Equal("some-event-name"))
			Expect(cfg.ProvidersEnv.GitHub.EventPath).To(Equal("some-event-path"))
			Expect(cfg.ProvidersEnv.GitHub.HeadRef).To(Equal("main"))
		})
	})

	Context("with Buildkite environment variables", func() {
		BeforeEach(func() {
			helpers.SetEnvFromFile("../../test/.env.buildkite")
		})

		It("parses the Buildkite options", func() {
			cfg, err := captain.InitConfig(cmd, cliArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.ProvidersEnv.Buildkite.Detected).To(BeTrue())
			Expect(cfg.ProvidersEnv.Buildkite.Branch).To(Equal("some-branch"))
			Expect(cfg.ProvidersEnv.Buildkite.BuildCreatorEmail).To(Equal("foo@bar.com"))
			Expect(cfg.ProvidersEnv.Buildkite.BuildID).To(Equal("123"))
			Expect(cfg.ProvidersEnv.Buildkite.BuildURL).To(Equal("https://buildkite.com/builds/123"))
			Expect(cfg.ProvidersEnv.Buildkite.Commit).To(Equal("1fc108cab0bb46083c6cdd50f8cd1deb5005e235"))
			Expect(cfg.ProvidersEnv.Buildkite.JobID).To(Equal("987"))
			Expect(cfg.ProvidersEnv.Buildkite.Label).To(Equal("Fake"))
			Expect(cfg.ProvidersEnv.Buildkite.Message).To(Equal("Fixed it"))
			Expect(cfg.ProvidersEnv.Buildkite.OrganizationSlug).To(Equal("rwx"))
			Expect(cfg.ProvidersEnv.Buildkite.Repo).To(Equal("https://github.com/rwx-research/captain-cli"))
			Expect(cfg.ProvidersEnv.Buildkite.RetryCount).To(Equal("0"))
			Expect(cfg.ProvidersEnv.Buildkite.ParallelJob).To(Equal("1"))
			Expect(cfg.ProvidersEnv.Buildkite.ParallelJobCount).To(Equal("2"))
		})
	})

	Context("with CircleCI environment variables", func() {
		BeforeEach(func() {
			helpers.SetEnvFromFile("../../test/.env.circleci")
		})

		It("parses the CircleCI options", func() {
			cfg, err := captain.InitConfig(cmd, cliArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.ProvidersEnv.CircleCI.Detected).To(BeTrue())
			Expect(cfg.ProvidersEnv.CircleCI.BuildNum).To(Equal("18"))
			Expect(cfg.ProvidersEnv.CircleCI.BuildURL).To(Equal("https://circleci.com/gh/michaelglass/rspec-abq-test/18"))
			Expect(cfg.ProvidersEnv.CircleCI.Job).To(Equal("build"))
			Expect(cfg.ProvidersEnv.CircleCI.NodeIndex).To(Equal("0"))
			Expect(cfg.ProvidersEnv.CircleCI.NodeTotal).To(Equal("2"))
			Expect(cfg.ProvidersEnv.CircleCI.ProjectReponame).To(Equal("rspec-abq-test"))
			Expect(cfg.ProvidersEnv.CircleCI.ProjectUsername).To(Equal("michaelglass"))
			Expect(cfg.ProvidersEnv.CircleCI.RepositoryURL).To(Equal("git@github.com:michaelglass/rspec-abq-test.git"))
			Expect(cfg.ProvidersEnv.CircleCI.Branch).To(Equal("circle-ci"))
			Expect(cfg.ProvidersEnv.CircleCI.Sha1).To(Equal("1fc108cab0bb46083c6cdd50f8cd1deb5005e235"))
			Expect(cfg.ProvidersEnv.CircleCI.Username).To(Equal("michaelglass"))
		})
	})

	Context("with GitLab environment variables", func() {
		BeforeEach(func() {
			helpers.SetEnvFromFile("../../test/.env.gitlab_ci")
		})

		It("parses the GitLab options", func() {
			cfg, err := captain.InitConfig(cmd, cliArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.ProvidersEnv.GitLab.Detected).To(BeTrue())
			Expect(cfg.ProvidersEnv.GitLab.JobName).To(Equal("rspec 1/2"))
			Expect(cfg.ProvidersEnv.GitLab.JobStage).To(Equal("test"))
			Expect(cfg.ProvidersEnv.GitLab.JobID).To(Equal("3889399980"))
			Expect(cfg.ProvidersEnv.GitLab.PipelineID).To(Equal("798778026"))
			Expect(cfg.ProvidersEnv.GitLab.JobURL).To(Equal("https://gitlab.com/captain-examples/rspec/-/jobs/3889399980"))
			Expect(cfg.ProvidersEnv.GitLab.PipelineURL).To(
				Equal("https://gitlab.com/captain-examples/rspec/-/pipelines/798778026"))
			Expect(cfg.ProvidersEnv.GitLab.NodeTotal).To(Equal("2"))
			Expect(cfg.ProvidersEnv.GitLab.NodeIndex).To(Equal("1"))
			Expect(cfg.ProvidersEnv.GitLab.UserLogin).To(Equal("michaelglass"))
			Expect(cfg.ProvidersEnv.GitLab.ProjectPath).To(Equal("captain-examples/rspec"))
			Expect(cfg.ProvidersEnv.GitLab.ProjectURL).To(Equal("https://gitlab.com/captain-examples/rspec"))
			Expect(cfg.ProvidersEnv.GitLab.CommitSHA).To(Equal("1fc108cab0bb46083c6cdd50f8cd1deb5005e235"))
			Expect(cfg.ProvidersEnv.GitLab.CommitAuthor).To(Equal("Michael Glass <me@mike.is>"))
			Expect(cfg.ProvidersEnv.GitLab.CommitBranch).To(Equal("main"))
			Expect(cfg.ProvidersEnv.GitLab.CommitMessage).To(Equal("print env"))
			Expect(cfg.ProvidersEnv.GitLab.APIV4URL).To(Equal("https://gitlab.com/api/v4"))
		})
	})

	Context("with Captain environment variables", func() {
		BeforeEach(func() {
			helpers.SetEnv(map[string]string{
				"CAPTAIN_WHO":            "tester",
				"CAPTAIN_SHA":            "1fc108cab0bb46083c6cdd50f8cd1deb5005e235",
				"CAPTAIN_BRANCH":         "main",
				"CAPTAIN_BUILD_URL":      "https://example.com/build/1",
				"CAPTAIN_TITLE":          "Captain Test",
				"CAPTAIN_COMMIT_MESSAGE": "print env",
			})
		})

		It("parses the Captain variables", func() {
			err := captain.AddFlags(cmd, &cliArgs)
			Expect(err).ToNot(HaveOccurred())

			cfg, err := captain.InitConfig(cmd, cliArgs)
			Expect(err).ToNot(HaveOccurred())

			Expect(cfg.ProvidersEnv.Generic.Who).To(Equal("tester"))
			Expect(cfg.ProvidersEnv.Generic.Sha).To(Equal("1fc108cab0bb46083c6cdd50f8cd1deb5005e235"))
			Expect(cfg.ProvidersEnv.Generic.Branch).To(Equal("main"))
			Expect(cfg.ProvidersEnv.Generic.BuildURL).To(Equal("https://example.com/build/1"))
			Expect(cfg.ProvidersEnv.Generic.Title).To(Equal("Captain Test"))
			Expect(cfg.ProvidersEnv.Generic.CommitMessage).To(Equal("print env"))
		})
	})
})
