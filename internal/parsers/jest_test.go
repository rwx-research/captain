package parsers_test

import (
	"os"
	"time"

	"github.com/rwx-research/captain-cli/internal/parsers"
	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Jest", func() {
	var (
		err     error
		fixture *os.File
		parser  *parsers.Jest
		result  map[string]testing.TestResult
	)

	BeforeEach(func() {
		fixture, err = os.Open("../../test/fixtures/jest.json")
		Expect(err).ToNot(HaveOccurred())

		parser = new(parsers.Jest)
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

		Expect(result).To(HaveLen(18), "total test count")
		Expect(failedTestCount).To(Equal(6), "failed tests count")
		Expect(pendingTestCount).To(Equal(6), "pending test count")
		Expect(successfulTestCount).To(Equal(6), "successful test count")
		Expect(unknownStatusCount).To(BeZero())
	})

	It("extracts the test metadata", func() {
		key := "/home/runner/work/captain/captain/app/javascript/controllers/top_level.test.js > is top-level passing"
		Expect(result).To(HaveKey(key))
		Expect(result[key].Description).To(Equal("is top-level passing"))
		Expect(result[key].Duration).To(Equal(time.Duration(1000000000)))
		Expect(result[key].Meta).To(Equal(
			map[string]any{"file": "/home/runner/work/captain/captain/app/javascript/controllers/top_level.test.js"},
		))
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
})
