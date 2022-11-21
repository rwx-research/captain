package testing_test

import (
	"time"

	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestPartition.Add", func() {
	It("appends matching test file path and updates remaining capacity", func() {
		partition := testing.TestPartition{
			RemainingCapacity: time.Duration(10),
			Index:             0,
			TestFilePaths:     []string{},
			TotalCapacity:     time.Duration(100),
		}
		testFileTiming := testing.TestFileTiming{
			Filepath: "./spec/a_spec.rb",
			Duration: time.Duration(2),
		}
		fileTimingMatch := testing.FileTimingMatch{
			FileTiming:     testFileTiming,
			ClientFilepath: "spec/a_spec.rb",
		}
		partition = partition.Add(fileTimingMatch)

		Expect(partition.RemainingCapacity).To(Equal(time.Duration(8)))
		Expect(partition.TestFilePaths).To(Equal([]string{"spec/a_spec.rb"}))
	})
})

var _ = Describe("TestPartition.AddFilePath", func() {
	It("appends filepath to the but doesn't update remaining capacity", func() {
		remainingCapacity := time.Duration(10)
		clientTestFilepath := "spec/a_spec.rb"
		partition := testing.TestPartition{
			RemainingCapacity: remainingCapacity,
			Index:             0,
			TestFilePaths:     []string{},
			TotalCapacity:     time.Duration(100),
		}
		partition = partition.AddFilePath(clientTestFilepath)

		Expect(partition.RemainingCapacity).To(Equal(remainingCapacity))
		Expect(partition.TestFilePaths).To(Equal([]string{clientTestFilepath}))
	})
})
