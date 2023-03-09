package main_test

import (
	"bufio"
	"os"
	"strings"

	"github.com/spf13/cobra"

	captain "github.com/rwx-research/captain-cli/cmd/captain"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("InitConfig", func() {
	var (
		cfg            captain.Config
		cmd            *cobra.Command
		err            error
		setEnvFromFile func(string)
	)

	BeforeEach(func() {
		envPrefixes := []string{"GITHUB", "BUILDKITE", "CIRCLE", "GITLAB", "CI", "RWX", "CAPTAIN"}
		for _, env := range os.Environ() {
			for _, prefix := range envPrefixes {
				if strings.HasPrefix(env, prefix) {
					pair := strings.SplitN(env, "=", 2)
					os.Unsetenv(pair[0])
				}
			}
		}

		setEnvFromFile = func(fileName string) {
			fd, err := os.Open(fileName)
			Expect(err).ToNot(HaveOccurred())
			defer fd.Close()

			scanner := bufio.NewScanner(fd)
			for scanner.Scan() {
				line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "export"))
				fields := strings.SplitN(line, "=", 2)
				os.Setenv(fields[0], fields[1])
			}
		}
	})

	JustBeforeEach(func() {
		cfg, err = captain.InitConfig(cmd)
	})

	Context("no environment variables", func() {
		It("uses cobra's default values", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.TestSuites).To(HaveKey(""))
			Expect(cfg.TestSuites[""].Retries.Attempts).To(Equal(-1))
			Expect(cfg.TestSuites[""].Retries.FlakyAttempts).To(Equal(-1))
		})
	})

	Context("with GitHub Actions environment variables", func() {
		BeforeEach(func() {
			setEnvFromFile("../../test/.env.github.actions")
		})

		It("parses the GitHub options", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.GitHub.Detected).To(BeTrue())
			Expect(cfg.GitHub.Repository).To(Equal("rwx-research/captain-cli"))
			Expect(cfg.GitHub.Name).To(Equal("some-job-name"))
			Expect(cfg.GitHub.Attempt).To(Equal("4"))
			Expect(cfg.GitHub.ID).To(Equal("1234"))
			Expect(cfg.GitHub.CommitSha).To(Equal("some-sha-for-testing"))
			Expect(cfg.GitHub.ExecutingActor).To(Equal("captain-cli-test"))
			Expect(cfg.GitHub.TriggeringActor).To(Equal("captain-cli-test-trigger"))
			Expect(cfg.GitHub.EventName).To(Equal("some-event-name"))
			Expect(cfg.GitHub.EventPath).To(Equal("some-event-path"))
			Expect(cfg.GitHub.HeadRef).To(Equal("main"))
		})
	})

	Context("with Buildkite environment variables", func() {
		BeforeEach(func() {
			setEnvFromFile("../../test/.env.buildkite")
		})

		It("parses the Buildkite options", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Buildkite.Detected).To(BeTrue())
			Expect(cfg.Buildkite.Branch).To(Equal("some-branch"))
			Expect(cfg.Buildkite.BuildCreatorEmail).To(Equal("foo@bar.com"))
			Expect(cfg.Buildkite.BuildID).To(Equal("123"))
			Expect(cfg.Buildkite.BuildURL).To(Equal("https://buildkite.com/builds/123"))
			Expect(cfg.Buildkite.Commit).To(Equal("abc123"))
			Expect(cfg.Buildkite.JobID).To(Equal("987"))
			Expect(cfg.Buildkite.Label).To(Equal("Fake"))
			Expect(cfg.Buildkite.Message).To(Equal("\"Fixed it\""))
			Expect(cfg.Buildkite.OrganizationSlug).To(Equal("rwx"))
			Expect(cfg.Buildkite.Repo).To(Equal("https://github.com/rwx-research/captain-cli"))
			Expect(cfg.Buildkite.RetryCount).To(Equal("0"))
			Expect(cfg.Buildkite.ParallelJob).To(Equal("1"))
			Expect(cfg.Buildkite.ParallelJobCount).To(Equal("2"))
		})
	})

	Context("with CircleCI environment variables", func() {
		BeforeEach(func() {
			setEnvFromFile("../../test/.env.circleci")
		})

		It("parses the CircleCI options", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.CircleCI.Detected).To(BeTrue())
			Expect(cfg.CircleCI.BuildNum).To(Equal("18"))
			Expect(cfg.CircleCI.BuildURL).To(Equal("https://circleci.com/gh/michaelglass/rspec-abq-test/18"))
			Expect(cfg.CircleCI.Job).To(Equal("build"))
			Expect(cfg.CircleCI.NodeIndex).To(Equal("0"))
			Expect(cfg.CircleCI.NodeTotal).To(Equal("2"))
			Expect(cfg.CircleCI.ProjectReponame).To(Equal("rspec-abq-test"))
			Expect(cfg.CircleCI.ProjectUsername).To(Equal("michaelglass"))
			Expect(cfg.CircleCI.RepositoryURL).To(Equal("git@github.com:michaelglass/rspec-abq-test.git"))
			Expect(cfg.CircleCI.Branch).To(Equal("circle-ci"))
			Expect(cfg.CircleCI.Sha1).To(Equal("000bd5713d35f778fb51d2b0bf034e8fff5b60b1"))
			Expect(cfg.CircleCI.Username).To(Equal("michaelglass"))
		})
	})

	Context("with GitLab environment variables", func() {
		BeforeEach(func() {
			setEnvFromFile("../../test/.env.gitlab_ci")
		})

		It("parses the GitLab options", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.GitLab.Detected).To(BeTrue())
			Expect(cfg.GitLab.CIJobName).To(Equal("rspec 1/2"))
			Expect(cfg.GitLab.CIJobStage).To(Equal("test"))
			Expect(cfg.GitLab.CIJobID).To(Equal("3889399980"))
			Expect(cfg.GitLab.CIPipelineID).To(Equal("798778026"))
			Expect(cfg.GitLab.CIJobURL).To(Equal("https://gitlab.com/captain-examples/rspec/-/jobs/3889399980"))
			Expect(cfg.GitLab.CIPipelineURL).To(Equal("https://gitlab.com/captain-examples/rspec/-/pipelines/798778026"))
			Expect(cfg.GitLab.CINodeTotal).To(Equal("2"))
			Expect(cfg.GitLab.CINodeIndex).To(Equal("1"))
			Expect(cfg.GitLab.GitlabUserLogin).To(Equal("michaelglass"))
			Expect(cfg.GitLab.CIProjectPath).To(Equal("captain-examples/rspec"))
			Expect(cfg.GitLab.CIProjectURL).To(Equal("https://gitlab.com/captain-examples/rspec"))
			Expect(cfg.GitLab.CICommitSHA).To(Equal("03d68a49ef1e131cf8942d5a07a0ff008ede6a1a"))
			Expect(cfg.GitLab.CICommitAuthor).To(Equal("Michael Glass <me@mike.is>"))
			Expect(cfg.GitLab.CICommitBranch).To(Equal("main"))
			Expect(cfg.GitLab.CICommitMessage).To(Equal("print env"))
			Expect(cfg.GitLab.CIAPIV4URL).To(Equal("https://gitlab.com/api/v4"))
		})
	})
})
