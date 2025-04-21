package testing_test

import (
	"time"

	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestPartition.Add", func() {
	It("appends matching test file path and updates the runtime", func() {
		partition := testing.TestPartition{
			Index:         0,
			TestFilePaths: []string{},
			Runtime:       time.Duration(10),
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

		Expect(partition.Runtime).To(Equal(time.Duration(12)))
		Expect(partition.TestFilePaths).To(Equal([]string{"spec/a_spec.rb"}))
	})
})

var _ = Describe("TestPartition.AddFilePath", func() {
	It("appends filepath to the but doesn't update the runtime", func() {
		clientTestFilepath := "spec/a_spec.rb"
		partition := testing.TestPartition{
			Index:         0,
			TestFilePaths: []string{},
			Runtime:       time.Duration(10),
		}
		partition = partition.AddFilePath(clientTestFilepath)

		Expect(partition.Runtime).To(Equal(time.Duration(10)))
		Expect(partition.TestFilePaths).To(Equal([]string{clientTestFilepath}))
	})
})
