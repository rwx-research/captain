package parsers_test

import (
	"os"
	"time"

	"github.com/rwx-research/captain-cli/internal/parsers"
	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Junit", func() {
	var (
		err     error
		fixture *os.File
		parser  *parsers.JUnit
		result  map[string]testing.TestResult
	)

	BeforeEach(func() {
		fixture, err = os.Open("../../test/fixtures/junit.xml")
		Expect(err).ToNot(HaveOccurred())

		parser = new(parsers.JUnit)
	})

	JustBeforeEach(func() {
		result, err = parser.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())
	})

	It("detects test statuses", func() {
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

		Expect(result).To(HaveLen(71), "total test count")
		Expect(failedTestCount).To(Equal(3), "failed tests count")
		Expect(pendingTestCount).To(Equal(2), "pending test count")
		Expect(successfulTestCount).To(Equal(66), "successful test count")
		Expect(unknownStatusCount).To(BeZero())
	})

	It("extracts the test metadata", func() {
		key := "reporting::test_dot_reporter::breaks_lines_with_many_dots"
		Expect(result).To(HaveKey(key))
		Expect(result[key].Description).To(Equal("reporting::test_dot_reporter::breaks_lines_with_many_dots"))
		Expect(result[key].Duration).To(Equal(time.Duration(352000000)))
		Expect(result[key].Meta).To(Equal(map[string]any{}))
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
