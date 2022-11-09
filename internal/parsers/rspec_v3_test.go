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
		result  []testing.TestResult
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
		key := "Tests::Case has top-level passing tests"

		for _, testResult := range result {
			if testResult.Description != key {
				continue
			}

			Expect(testResult.Duration).To(Equal(time.Duration(30795000)))
			Expect(testResult.Meta).To(Equal(
				map[string]any{"id": "./spec/examples/class_spec.rb[1:1]", "file": "./spec/examples/class_spec.rb"},
			))
			return
		}

		Expect(true).To(Equal(false), "Unreachable")
	})

	It("extracts the test metadata in absence of an id", func() {
		key := "some string within a context behaves like shared examples has top-level passing tests"

		for _, testResult := range result {
			if testResult.Description != key {
				continue
			}

			Expect(testResult.Duration).To(Equal(time.Duration(1960000)))
			Expect(testResult.Meta).To(Equal(
				map[string]any{
					"id":   "some string within a context behaves like shared examples has top-level passing tests",
					"file": "./spec/examples/shared_examples.rb",
				},
			))
			return
		}

		Expect(true).To(Equal(false), "Unreachable")
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
