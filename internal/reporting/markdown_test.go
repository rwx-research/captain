package reporting_test

import (
	"strings"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/reporting"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Markdown Report", func() {
	var (
		mockFile    *mocks.File
		testResults v1.TestResults
	)

	BeforeEach(func() {
		mockFile = new(mocks.File)
		mockFile.Builder = new(strings.Builder)

		message := "expected true to equal false"
		fifteen := 15
		testResults = *v1.NewTestResults(
			v1.RubyRSpecFramework,
			[]v1.Test{
				{
					Name:    "successful test",
					Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
				{
					Name:     "failed test",
					Location: &v1.Location{File: "some/path/to/file.rb"},
					Attempt: v1.TestAttempt{
						Status: v1.NewFailedTestStatus(
							&message,
							nil,
							[]string{"file/path/one.rb:4", "file/path/two.rb:4", "file/path/three.rb:4"},
						),
					},
					PastAttempts: []v1.TestAttempt{
						{Status: v1.NewFailedTestStatus(nil, nil, nil)},
						{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
				},
				{
					Name:     "other failed test",
					Location: &v1.Location{File: "some/path/to/file.rb"},
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
				},
				{
					Name:     "flaky test",
					Location: &v1.Location{File: "some/path/to/file.rb", Line: &fifteen},
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					PastAttempts: []v1.TestAttempt{
						{
							Status: v1.NewFailedTestStatus(
								&message,
								nil,
								[]string{"file/path/one.rb:4", "file/path/two.rb:4", "file/path/three.rb:4"},
							),
						},
					},
				},
				{
					Name:    "skipped test",
					Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
				},
				{
					Name:    "timed out test",
					Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
				},
			},
			nil,
		)
	})

	It("produces a readable summary", func() {
		Expect(reporting.WriteMarkdownSummary(mockFile, testResults, reporting.Configuration{})).To(Succeed())
		summary := mockFile.Builder.String()
		cupaloy.SnapshotT(GinkgoT(), summary)
	})

	It("produces a truncated summary <= 1MB", func() {
		hundredKBName := new(strings.Builder)
		for i := 0; i < 100000; i++ {
			_, err := hundredKBName.WriteString("a")
			Expect(err).NotTo(HaveOccurred())
		}

		tests := make([]v1.Test, 0)
		for i := 0; i < 100; i++ {
			tests = append(tests, v1.Test{
				Name:    hundredKBName.String(),
				Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
			})
		}

		Expect(
			reporting.WriteMarkdownSummary(
				mockFile,
				*v1.NewTestResults(v1.RubyRSpecFramework, tests, nil),
				reporting.Configuration{},
			),
		).To(Succeed())
		summary := mockFile.Builder.String()
		cupaloy.SnapshotT(GinkgoT(), summary)

		Expect(len(summary) < 1000000).To(Equal(true))
		Expect(strings.Count(summary, hundredKBName.String()) < 10).To(Equal(true))
	})
})
