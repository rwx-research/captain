package reporting_test

import (
	"encoding/xml"
	"strings"

	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/reporting"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JUnit Report", func() {
	var (
		mockFile    *mocks.File
		testResults v1.TestResults
	)

	BeforeEach(func() {
		mockFile = new(mocks.File)
		mockFile.Builder = new(strings.Builder)

		message := "expected true to equal false"
		messageWithAnsi := `[31mFailure/Error: [0m[32mexpect[0m(thanos).to ` +
			`eq([31m[1;31m"[0m[31minevitable[1;31m"[0m[31m[0m)[0m
[31m[0m
[31m  expected: "inevitable"[0m
[31m       got: "evitable"[0m
[31m[0m
[31m  (compared using ==)[0m`

		testResults = v1.TestResults{
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
					Location: &v1.Location{
						File: "/path/to/file",
						Line: new(int),
					},
				},
				{
					Name: "failed test message only w/o ansi",
					Attempt: v1.TestAttempt{
						Status: v1.NewFailedTestStatus(
							&message,
							nil,
							nil,
						),
					},
				},
				{
					Name: "failed test message only w/ ansi",
					Attempt: v1.TestAttempt{
						Status: v1.NewFailedTestStatus(
							&messageWithAnsi,
							nil,
							nil,
						),
					},
				},
			},
		}
	})

	It("produces a parsable JUnit file", func() {
		var result parsing.JUnitTestResults

		Expect(reporting.WriteJUnitSummary(mockFile, testResults, reporting.Configuration{})).To(Succeed())
		Expect(xml.Unmarshal([]byte(mockFile.Builder.String()), &result)).To(Succeed())
		Expect(result.TestSuites).To(HaveLen(1))

		Expect(result.TestSuites[0].Errors).To(Equal(12))
		Expect(result.TestSuites[0].Failures).To(Equal(9))
		Expect(result.TestSuites[0].Skipped).To(Equal(25))

		Expect(result.TestSuites[0].TestCases).To(HaveLen(3))
		Expect(result.TestSuites[0].TestCases[0].Name).To(Equal("name of the test"))
		Expect(*result.TestSuites[0].TestCases[0].File).To(Equal("/path/to/file"))
		Expect(*result.TestSuites[0].TestCases[0].Line).To(Equal(0))

		Expect(result.TestSuites[0].TestCases[1].Name).To(Equal("failed test message only w/o ansi"))
		Expect(*result.TestSuites[0].TestCases[1].Failure.Message).To(Equal("expected true to equal false"))

		Expect(result.TestSuites[0].TestCases[2].Name).To(Equal("failed test message only w/ ansi"))
		Expect(*result.TestSuites[0].TestCases[2].Failure.Message).To(Equal(`Failure/Error: expect(thanos).to eq("inevitable")

  expected: "inevitable"
       got: "evitable"

  (compared using ==)`))
	})
})
