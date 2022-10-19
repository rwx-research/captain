package parsers_test

import (
	"os"

	"github.com/rwx-research/captain-cli/internal/parsers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Jest", func() {
	var (
		fixture *os.File
		parser  *parsers.Jest
	)

	BeforeEach(func() {
		var err error
		fixture, err = os.Open("../../test/fixtures/jest.json")
		Expect(err).ToNot(HaveOccurred())

		parser = new(parsers.Jest)
	})

	JustBeforeEach(func() {
		Expect(parser.Parse(fixture)).To(Succeed())
	})

	It("detects successful & failed tests", func() {
		var failedTestCount, successfulTestCount int

		for parser.NextTestCase() {
			if parser.IsTestCaseFailed() {
				failedTestCount++
			} else {
				successfulTestCount++
			}
		}

		Expect(failedTestCount+successfulTestCount).To(Equal(18), "total test count")
		Expect(failedTestCount).To(Equal(6), "failed tests count")
		Expect(successfulTestCount).To(Equal(12), "successful test count")
	})

	It("extracts the test name", func() {
		Expect(parser.NextTestCase())
		Expect(parser.TestCaseID()).To(Equal(
			"/home/runner/work/captain/captain/app/javascript/controllers/top_level.test.js > is top-level pending",
		))
	})
})
