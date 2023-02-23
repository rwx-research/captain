package reporting_test

import (
	"encoding/xml"
	"strings"

	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/reporting"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JUnit Report", func() {
	var (
		mockFile    *mocks.File
		testResults []v1.TestResults
	)

	BeforeEach(func() {
		mockFile = new(mocks.File)
		mockFile.Builder = new(strings.Builder)

		testResults = []v1.TestResults{{
			Framework: v1.Framework{
				Language: "Ruby",
				Kind:     "RSpec",
			},
			Summary: v1.Summary{
				Status:      v1.SummaryStatusSuccessful,
				Tests:       1,
				OtherErrors: 2,
				Retries:     3,
				Canceled:    4,
				Failed:      5,
				Pended:      6,
				Quarantined: 7,
				Skipped:     8,
				Successful:  9,
				TimedOut:    10,
				Todo:        11,
			},
			Tests: []v1.Test{
				{
					Name: "name of the test",
					Attempt: v1.TestAttempt{
						Duration: nil,
						Status:   v1.TestStatus{Kind: "successful"},
					},
				},
			},
		}}
	})

	It("produces a parsable JUnit file", func() {
		var result parsing.JUnitTestResults

		Expect(reporting.WriteJUnitSummary(mockFile, testResults)).To(Succeed())
		Expect(xml.Unmarshal([]byte(mockFile.Builder.String()), &result)).To(Succeed())
		Expect(result.TestSuites).To(HaveLen(len(testResults)))

		Expect(result.TestSuites[0].Errors).To(Equal(12))
		Expect(result.TestSuites[0].Failures).To(Equal(9))
		Expect(result.TestSuites[0].Skipped).To(Equal(25))

		Expect(result.TestSuites[0].TestCases).To(HaveLen(1))
		Expect(result.TestSuites[0].TestCases[0].Name).To(Equal("name of the test"))
	})
})
