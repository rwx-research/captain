package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/config"
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MintEnv.MakeProvider", func() {
	var env providers.Env

	BeforeEach(func() {
		partitionIndex := "0"
		partitionTotal := "2"
		actorID := "some-actor-id"
		env = providers.Env{Mint: providers.MintEnv{
			Detected:          true,
			ParallelIndex:     &partitionIndex,
			ParallelTotal:     &partitionTotal,
			ActorID:           &actorID,
			Actor:             "some-actor",
			RunURL:            "https://cloud.rwx.com/mint/some-org/runs/some-run-id",
			TaskURL:           "https://cloud.rwx.com/mint/some-org/tasks/some-task-id",
			RunID:             "some-run-id",
			TaskID:            "some-task-id",
			TaskAttemptNumber: "1",
			RunTitle:          "Some title",
			GitRepositoryURL:  "git@github.com:rwx-research/captain.git",
			GitRepositoryName: "rwx-research/captain",
			GitCommitMessage:  "Some commit message",
			GitCommitSha:      "some-commit-sha",
			GitRef:            "refs/heads/some-ref",
			GitRefName:        "some-ref",
		}}
	})

	It("is valid", func() {
		provider, err := env.MakeProvider()
		Expect(err).To(BeNil())
		Expect(provider.AttemptedBy).To(Equal("some-actor"))
		Expect(provider.BranchName).To(Equal("some-ref"))
		Expect(provider.CommitSha).To(Equal("some-commit-sha"))
		Expect(provider.CommitMessage).To(Equal("Some commit message"))
		Expect(provider.ProviderName).To(Equal("mint"))
		Expect(provider.Title).To(Equal("Some title"))
		Expect(provider.PartitionNodes).To(Equal(config.PartitionNodes{Index: 0, Total: 2}))
	})

	It("requires Actor", func() {
		env.Mint.Actor = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing actor"))
	})
	It("requires RunURL", func() {
		env.Mint.RunURL = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing run url"))
	})
	It("requires TaskURL", func() {
		env.Mint.TaskURL = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing task url"))
	})
	It("requires RunID", func() {
		env.Mint.RunID = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing run id"))
	})
	It("requires TaskID", func() {
		env.Mint.TaskID = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing task id"))
	})
	It("requires TaskAttemptNumber", func() {
		env.Mint.TaskAttemptNumber = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing task attempt number"))
	})
	It("requires RunTitle", func() {
		env.Mint.RunTitle = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing run title"))
	})
	It("requires GitRepositoryURL", func() {
		env.Mint.GitRepositoryURL = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing git repository url"))
	})
	It("requires GitRepositoryName", func() {
		env.Mint.GitRepositoryName = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing git repository name"))
	})
	It("requires GitCommitMessage", func() {
		env.Mint.GitCommitMessage = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing git commit message"))
	})
	It("requires GitCommitSha", func() {
		env.Mint.GitCommitSha = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing git commit sha"))
	})
	It("requires GitRef", func() {
		env.Mint.GitRef = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing git ref"))
	})
	It("requires GitRefName", func() {
		env.Mint.GitRefName = ""
		_, err := env.MakeProvider()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing git ref name"))
	})
})

var _ = Describe("MintEnv.JobTags", func() {
	It("constructs job tags with optional attributes", func() {
		partitionIndex := "0"
		partitionTotal := "2"
		actorID := "some-actor-id"
		provider, _ := providers.Env{Mint: providers.MintEnv{
			Detected:          true,
			ParallelIndex:     &partitionIndex,
			ParallelTotal:     &partitionTotal,
			ActorID:           &actorID,
			Actor:             "some-actor",
			RunURL:            "https://cloud.rwx.com/mint/some-org/runs/some-run-id",
			TaskURL:           "https://cloud.rwx.com/mint/some-org/tasks/some-task-id",
			RunID:             "some-run-id",
			TaskID:            "some-task-id",
			TaskAttemptNumber: "1",
			RunTitle:          "Some title",
			GitRepositoryURL:  "git@github.com:rwx-research/captain.git",
			GitRepositoryName: "rwx-research/captain",
			GitCommitMessage:  "Some commit message",
			GitCommitSha:      "some-commit-sha",
			GitRef:            "refs/heads/some-ref",
			GitRefName:        "some-ref",
		}}.MakeProvider()

		Expect(provider.JobTags).To(Equal(map[string]any{
			"mint_actor":               "some-actor",
			"mint_run_url":             "https://cloud.rwx.com/mint/some-org/runs/some-run-id",
			"mint_task_url":            "https://cloud.rwx.com/mint/some-org/tasks/some-task-id",
			"mint_run_id":              "some-run-id",
			"mint_task_id":             "some-task-id",
			"mint_task_attempt_number": "1",
			"mint_run_title":           "Some title",
			"mint_git_repository_url":  "git@github.com:rwx-research/captain.git",
			"mint_git_repository_name": "rwx-research/captain",
			"mint_git_commit_message":  "Some commit message",
			"mint_git_commit_sha":      "some-commit-sha",
			"mint_git_ref":             "refs/heads/some-ref",
			"mint_git_ref_name":        "some-ref",
			"mint_parallel_index":      partitionIndex,
			"mint_parallel_total":      partitionTotal,
			"mint_actor_id":            actorID,
		}))
	})

	It("constructs job tags without optional attributes", func() {
		provider, _ := providers.Env{Mint: providers.MintEnv{
			Detected:          true,
			ParallelIndex:     nil,
			ParallelTotal:     nil,
			ActorID:           nil,
			Actor:             "some-actor",
			RunURL:            "https://cloud.rwx.com/mint/some-org/runs/some-run-id",
			TaskURL:           "https://cloud.rwx.com/mint/some-org/tasks/some-task-id",
			RunID:             "some-run-id",
			TaskID:            "some-task-id",
			TaskAttemptNumber: "1",
			RunTitle:          "Some title",
			GitRepositoryURL:  "git@github.com:rwx-research/captain.git",
			GitRepositoryName: "rwx-research/captain",
			GitCommitMessage:  "Some commit message",
			GitCommitSha:      "some-commit-sha",
			GitRef:            "refs/heads/some-ref",
			GitRefName:        "some-ref",
		}}.MakeProvider()

		Expect(provider.JobTags).To(Equal(map[string]any{
			"mint_actor":               "some-actor",
			"mint_run_url":             "https://cloud.rwx.com/mint/some-org/runs/some-run-id",
			"mint_task_url":            "https://cloud.rwx.com/mint/some-org/tasks/some-task-id",
			"mint_run_id":              "some-run-id",
			"mint_task_id":             "some-task-id",
			"mint_task_attempt_number": "1",
			"mint_run_title":           "Some title",
			"mint_git_repository_url":  "git@github.com:rwx-research/captain.git",
			"mint_git_repository_name": "rwx-research/captain",
			"mint_git_commit_message":  "Some commit message",
			"mint_git_commit_sha":      "some-commit-sha",
			"mint_git_ref":             "refs/heads/some-ref",
			"mint_git_ref_name":        "some-ref",
			"mint_parallel_index":      nil,
			"mint_parallel_total":      nil,
			"mint_actor_id":            nil,
		}))
	})
})
