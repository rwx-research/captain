package parsers_test

import (
	"os"
	"time"

	"github.com/rwx-research/captain-cli/internal/parsers"
	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RspecV3", func() {
	var (
		err     error
		fixture *os.File
		parser  *parsers.RSpecV3
		result  map[string]testing.TestResult
	)

	BeforeEach(func() {
		var err error
		fixture, err = os.Open("../../test/fixtures/rspec.json")
		Expect(err).ToNot(HaveOccurred())

		parser = new(parsers.RSpecV3)
	})

	JustBeforeEach(func() {
		result, err = parser.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())
	})

	It("detects successful & failed tests", func() {
		var failedTestCount, pendingTestCount, successfulTestCount, unknownStatusCount int

		for _, testResult := range result {
			switch testResult.Status {
			case testing.TestStatusSuccessful:
				successfulTestCount++
			case testing.TestStatusFailed:
				failedTestCount++
			case testing.TestStatusPending:
				pendingTestCount++
			case testing.TestStatusUnknown:
				unknownStatusCount++
			}
		}

		Expect(result).To(HaveLen(72), "total test count")
		Expect(failedTestCount).To(Equal(36), "failed tests count")
		Expect(pendingTestCount).To(Equal(24), "pending test count")
		Expect(successfulTestCount).To(Equal(12), "successful test count")
		Expect(unknownStatusCount).To(BeZero())
	})

	It("extracts the test metadata", func() {
		key := "./spec/examples/class_spec.rb[1:1]"
		Expect(result).To(HaveKey(key))
		Expect(result[key].Description).To(Equal("Tests::Case has top-level passing tests"))
		Expect(result[key].Duration).To(Equal(time.Duration(30795000)))
	})

	It("adds a status message to failed tests", func() {
		var failedTest testing.TestResult
		for _, example := range result {
			if example.Status == testing.TestStatusFailed {
				failedTest = example
				break
			}
		}
		Expect(failedTest).NotTo(Equal(testing.TestResult{}))
		Expect(failedTest.StatusMessage).NotTo(BeEmpty())
	})

	It("adds a status message to pending tests", func() {
		var pendingTest testing.TestResult
		for _, example := range result {
			if example.Status == testing.TestStatusPending {
				pendingTest = example
				break
			}
		}
		Expect(pendingTest).NotTo(Equal(testing.TestResult{}))
		Expect(pendingTest.StatusMessage).NotTo(BeEmpty())
	})
})
