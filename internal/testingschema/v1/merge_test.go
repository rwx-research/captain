package v1_test

import (
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merge", func() {
	var (
		rubyRSpec1      *v1.TestResults
		rubyRSpec2      *v1.TestResults
		javaScriptJest1 *v1.TestResults
		javaScriptJest2 *v1.TestResults
	)

	BeforeEach(func() {
		rubyRSpec1 = &v1.TestResults{
			DerivedFrom: []v1.OriginalTestResults{
				{OriginalFilePath: "path 1"},
				{OriginalFilePath: "path 2"},
			},
			Framework: v1.NewRubyRSpecFramework(),
			OtherErrors: []v1.OtherError{
				{Message: "other error 1"},
			},
			Summary: v1.Summary{
				Status:      v1.SummaryStatusFailed,
				Tests:       3,
				OtherErrors: 1,
				Successful:  1,
				Failed:      1,
				Skipped:     1,
			},
			Tests: []v1.Test{
				{Name: "test 1", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
				{Name: "test 2", Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
				{Name: "test 3", Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
			},
		}
		rubyRSpec2 = &v1.TestResults{
			DerivedFrom: []v1.OriginalTestResults{
				{OriginalFilePath: "path 3"},
			},
			Framework: v1.NewRubyRSpecFramework(),
			OtherErrors: []v1.OtherError{
				{Message: "other error 2"},
				{Message: "other error 3"},
			},
			Summary: v1.Summary{
				Status:      v1.SummaryStatusFailed,
				Tests:       3,
				OtherErrors: 2,
				Successful:  1,
				Failed:      1,
				Skipped:     1,
			},
			Tests: []v1.Test{
				{Name: "test 4", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
				{Name: "test 5", Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
				{Name: "test 6", Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
			},
		}
		javaScriptJest1 = &v1.TestResults{
			DerivedFrom: []v1.OriginalTestResults{
				{OriginalFilePath: "path 4"},
				{OriginalFilePath: "path 5"},
			},
			Framework:   v1.NewJavaScriptJestFramework(),
			OtherErrors: []v1.OtherError{},
			Summary: v1.Summary{
				Status:     v1.SummaryStatusSuccessful,
				Tests:      2,
				Successful: 1,
				Skipped:    1,
			},
			Tests: []v1.Test{
				{Name: "test 7", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
				{Name: "test 8", Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
			},
		}
		javaScriptJest2 = &v1.TestResults{
			DerivedFrom: []v1.OriginalTestResults{},
			Framework:   v1.NewJavaScriptJestFramework(),
			OtherErrors: []v1.OtherError{},
			Summary: v1.Summary{
				Status:     v1.SummaryStatusSuccessful,
				Tests:      1,
				Successful: 1,
			},
			Tests: []v1.Test{
				{Name: "test 9", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
			},
		}
	})

	It("does not merge results from different frameworks", func() {
		Expect(v1.Merge([]v1.TestResults{*rubyRSpec1, *javaScriptJest1})).To(Equal(
			[]v1.TestResults{
				*javaScriptJest1,
				*rubyRSpec1,
			},
		))
	})

	It("merges results from the same framework", func() {
		Expect(v1.Merge([]v1.TestResults{*rubyRSpec1, *javaScriptJest1, *rubyRSpec2, *javaScriptJest2})).To(Equal(
			[]v1.TestResults{
				{
					DerivedFrom: []v1.OriginalTestResults{
						{OriginalFilePath: "path 4"},
						{OriginalFilePath: "path 5"},
					},
					Framework:   v1.NewJavaScriptJestFramework(),
					OtherErrors: []v1.OtherError{},
					Summary: v1.Summary{
						Status:     v1.SummaryStatusSuccessful,
						Tests:      3,
						Successful: 2,
						Skipped:    1,
					},
					Tests: []v1.Test{
						{Name: "test 7", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
						{Name: "test 8", Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
						{Name: "test 9", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
					},
				},
				{
					DerivedFrom: []v1.OriginalTestResults{
						{OriginalFilePath: "path 1"},
						{OriginalFilePath: "path 2"},
						{OriginalFilePath: "path 3"},
					},
					Framework: v1.NewRubyRSpecFramework(),
					OtherErrors: []v1.OtherError{
						{Message: "other error 1"},
						{Message: "other error 2"},
						{Message: "other error 3"},
					},
					Summary: v1.Summary{
						Status:      v1.SummaryStatusFailed,
						Tests:       6,
						OtherErrors: 3,
						Successful:  2,
						Failed:      2,
						Skipped:     2,
					},
					Tests: []v1.Test{
						{Name: "test 1", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
						{Name: "test 2", Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
						{Name: "test 3", Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
						{Name: "test 4", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
						{Name: "test 5", Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
						{Name: "test 6", Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
					},
				},
			},
		))
	})
})
