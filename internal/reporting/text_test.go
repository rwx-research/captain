package reporting_test

import (
	"strings"

	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/reporting"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Text Report", func() {
	var (
		mockFile    *mocks.File
		testResults v1.TestResults
	)

	BeforeEach(func() {
		mockFile = new(mocks.File)
		mockFile.Builder = new(strings.Builder)

		testResults = v1.TestResults{
			Summary: v1.Summary{
				Status:     v1.SummaryStatusFailed,
				Tests:      4,
				Failed:     1,
				Skipped:    1,
				Successful: 1,
				TimedOut:   1,
			},
			Tests: []v1.Test{
				{
					Name: "successful test",
					Attempt: v1.TestAttempt{
						Duration: nil,
						Status:   v1.TestStatus{Kind: v1.TestStatusSuccessful},
					},
				},
				{
					Name: "failed test",
					Attempt: v1.TestAttempt{
						Duration: nil,
						Status:   v1.TestStatus{Kind: v1.TestStatusFailed},
					},
				},
				{
					Name: "skipped test",
					Attempt: v1.TestAttempt{
						Duration: nil,
						Status:   v1.TestStatus{Kind: v1.TestStatusSkipped},
					},
				},
				{
					Name: "timed out test",
					Attempt: v1.TestAttempt{
						Duration: nil,
						Status:   v1.TestStatus{Kind: v1.TestStatusTimedOut},
					},
				},
			},
		}
	})

	It("produces a readable summary", func() {
		Expect(reporting.WriteTextSummary(mockFile, testResults, reporting.Configuration{})).To(Succeed())
		summary := mockFile.Builder.String()

		Expect(summary).To(ContainSubstring("total of 4 tests"))
		Expect(summary).To(ContainSubstring("Failed (1)"))
		Expect(summary).To(ContainSubstring("Skipped (1)"))
		Expect(summary).To(ContainSubstring("TimedOut (1)"))
	})
})
