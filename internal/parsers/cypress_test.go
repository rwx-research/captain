package parsers_test

import (
	"os"
	"time"

	"github.com/rwx-research/captain-cli/internal/parsers"
	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cypress", func() {
	var (
		err     error
		fixture *os.File
		parser  *parsers.JUnit
		result  map[string]testing.TestResult
	)

	BeforeEach(func() {
		var err error

		fixture, err = os.Open("../../test/fixtures/cypress.xml")
		Expect(err).ToNot(HaveOccurred())

		parser = new(parsers.JUnit)
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

		Expect(result).To(HaveLen(11), "total test count")
		Expect(failedTestCount).To(Equal(3), "failed tests count")
		Expect(pendingTestCount).To(Equal(2), "pending test count")
		Expect(successfulTestCount).To(Equal(6), "successful test count")
		Expect(unknownStatusCount).To(BeZero())
	})

	It("extracts the test metadata", func() {
		key := "Login Flow When you are logged out and visit an authenticated path, you are redirected to the " +
			"authenticated path after login"
		Expect(result).To(HaveKey(key))
		Expect(result[key].Description).To(Equal(
			"Login Flow When you are logged out and visit an authenticated path, you are redirected to the authenticated " +
				"path after login",
		))
		Expect(result[key].Duration).To(Equal(time.Duration(10841000000)))
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
