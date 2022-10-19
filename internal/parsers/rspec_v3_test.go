package parsers_test

import (
	"os"

	"github.com/rwx-research/captain-cli/internal/parsers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RspecV3", func() {
	var (
		fixture *os.File
		parser  *parsers.RSpecV3
	)

	BeforeEach(func() {
		var err error
		fixture, err = os.Open("../../test/fixtures/rspec.json")
		Expect(err).ToNot(HaveOccurred())

		parser = new(parsers.RSpecV3)
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

		Expect(failedTestCount+successfulTestCount).To(Equal(72), "total test count")
		Expect(failedTestCount).To(Equal(36), "failed tests count")
		Expect(successfulTestCount).To(Equal(36), "successful test count")
	})

	It("extracts the test name", func() {
		Expect(parser.NextTestCase())
		Expect(parser.TestCaseID()).To(Equal("./spec/examples/class_spec.rb[1:1]"))
	})
})
