package reporting_test

import (
	"strconv"
	"strings"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/providers"
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

		id1 := "./spec/foo/bar.rb[1:2:3]"
		id2 := "./spec/foo/bar.rb[4:5:6]"
		id3 := "./spec/foo/bar.rb[7:8:9]"
		id4 := "./spec/foo/bar.rb:12"
		id5 := "./spec/foo/bar.rb:11"
		id6 := "./spec/foo/bar.rb:12"
		id7 := "./spec/foo/bar.rb:13"
		id8 := "./spec/foo/bar.rb:14"
		message := "expected true to equal false"
		fifteen := 15
		testResults = *v1.NewTestResults(
			v1.RubyRSpecFramework,
			[]v1.Test{
				{
					ID:      &id1,
					Name:    "successful test",
					Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
				{
					ID:       &id2,
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
					ID:       &id3,
					Name:     "other failed test",
					Location: &v1.Location{File: "some/path/to/file.rb"},
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
				},
				{
					ID:       &id4,
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
					ID:      &id5,
					Name:    "skipped test",
					Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
				},
				{
					ID:      &id6,
					Name:    "timed out test",
					Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus()},
				},
				{
					ID:      &id7,
					Name:    "quarantined test",
					Attempt: v1.TestAttempt{Status: v1.NewQuarantinedTestStatus(v1.NewTimedOutTestStatus())},
				},
				{
					ID:      &id8,
					Name:    "canceled test",
					Attempt: v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
				},
			},
			nil,
		)
	})

	It("produces a readable summary when cloud is enabled", func() {
		cfg := reporting.Configuration{
			SuiteID:      "some-suite-id",
			CloudEnabled: true,
			CloudHost:    "example.com",
			Provider: providers.Provider{
				BranchName: "some/branch",
				CommitSha:  "abcdef113131",
			},
		}
		Expect(reporting.WriteMarkdownSummary(mockFile, testResults, cfg)).To(Succeed())
		summary := mockFile.Builder.String()
		cupaloy.SnapshotT(GinkgoT(), summary)
	})

	It("produces a readable summary with a custom retry template", func() {
		cfg := reporting.Configuration{
			SuiteID:              "some-suite-id",
			CloudEnabled:         false,
			CloudHost:            "",
			RetryCommandTemplate: "bin/rspec {{ tests }}",
			Provider:             providers.Provider{},
		}
		Expect(reporting.WriteMarkdownSummary(mockFile, testResults, cfg)).To(Succeed())
		summary := mockFile.Builder.String()
		cupaloy.SnapshotT(GinkgoT(), summary)
	})

	It("produces a readable summary when cloud is disabled", func() {
		cfg := reporting.Configuration{
			SuiteID:      "some-suite-id",
			CloudEnabled: false,
			CloudHost:    "",
			Provider:     providers.Provider{},
		}
		Expect(reporting.WriteMarkdownSummary(mockFile, testResults, cfg)).To(Succeed())
		summary := mockFile.Builder.String()
		cupaloy.SnapshotT(GinkgoT(), summary)
	})

	It("produces a truncated summary <= 1MB", func() {
		cfg := reporting.Configuration{
			SuiteID:      "some-suite-id",
			CloudEnabled: false,
			CloudHost:    "",
			Provider:     providers.Provider{},
		}

		hundredKBName := new(strings.Builder)
		for i := 0; i < 100000; i++ {
			_, err := hundredKBName.WriteString("a")
			Expect(err).NotTo(HaveOccurred())
		}

		tests := make([]v1.Test, 0)
		for i := 0; i < 100; i++ {
			id := strconv.Itoa(i)
			tests = append(tests, v1.Test{
				ID:      &id,
				Name:    hundredKBName.String(),
				Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
			})
		}

		Expect(
			reporting.WriteMarkdownSummary(
				mockFile,
				*v1.NewTestResults(v1.RubyRSpecFramework, tests, nil),
				cfg,
			),
		).To(Succeed())
		summary := mockFile.Builder.String()
		cupaloy.SnapshotT(GinkgoT(), summary)

		Expect(len(summary) < 1000000).To(Equal(true))
		Expect(strings.Count(summary, hundredKBName.String()) < 10).To(Equal(true))
	})
})
