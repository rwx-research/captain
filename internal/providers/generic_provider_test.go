package providers_test

import (
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MakeGenericProvider", func() {
	It("constructs job tags", func() {
		provider := providers.MakeGenericProvider(providers.GenericEnv{
			Who:            "test",
			Branch:         "testing-env-vars",
			Sha:            "abc123",
			CommitMessage:  "Testing env vars -- the commit message",
			BuildID:        "123",
			JobID:          "456",
			PartitionNodes: cli.PartitionNodes{Index: 0, Total: 2},
			JobName:        "rspec",
		})

		Expect(provider.JobTags).To(Equal(map[string]any{
			"captain_build_id":        "123",
			"captain_job_name":        "rspec",
			"captain_job_id":          "456",
			"captain_partition_index": 0,
			"captain_partition_total": 2,
		}))
	})
})
