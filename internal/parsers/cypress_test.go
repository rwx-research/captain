package parsers_test

import (
	"os"

	"github.com/rwx-research/captain-cli/internal/parsers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cypress", func() {
	var (
		fixture *os.File
		parser  *parsers.JUnit
	)

	BeforeEach(func() {
		var err error

		fixture, err = os.Open("../../test/fixtures/cypress.xml")
		Expect(err).ToNot(HaveOccurred())

		parser = new(parsers.JUnit)
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

		Expect(failedTestCount+successfulTestCount).To(Equal(11), "total test count")
		Expect(failedTestCount).To(Equal(3), "failed tests count")
		Expect(successfulTestCount).To(Equal(8), "successful test count")
	})

	It("extracts the test name", func() {
		Expect(parser.NextTestCase())
		Expect(parser.TestCaseID()).To(Equal(
			"Login Flow When you are logged out and visit an authenticated path, you are redirected to the authenticated path " +
				"after login",
		))
	})
})
