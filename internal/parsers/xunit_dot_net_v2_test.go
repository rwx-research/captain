package parsers_test

import (
	"os"

	"github.com/rwx-research/captain-cli/internal/parsers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("XunitDotNetV2", func() {
	var (
		fixture *os.File
		parser  *parsers.XUnitDotNetV2
	)

	BeforeEach(func() {
		var err error
		fixture, err = os.Open("../../test/fixtures/xunit_dot_net.xml")
		Expect(err).ToNot(HaveOccurred())

		parser = new(parsers.XUnitDotNetV2)
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

		Expect(failedTestCount+successfulTestCount).To(Equal(15), "total test count")
		Expect(failedTestCount).To(Equal(2), "failed tests count")
		Expect(successfulTestCount).To(Equal(13), "successful test count")
	})

	It("extracts the test name", func() {
		Expect(parser.NextTestCase())
		Expect(parser.TestCaseID()).To(Equal("test.xunit.assert.dll > NullAssertsTests+Null.Success"))
	})
})
