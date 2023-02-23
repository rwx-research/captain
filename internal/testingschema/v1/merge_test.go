package v1_test

import (
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merge", func() {
	var (
		rubyRSpec1   *v1.TestResults
		rubyRSpec1_2 *v1.TestResults
		rubyRSpec2   *v1.TestResults
	)

	BeforeEach(func() {
		rubyRSpec1 = &v1.TestResults{
			DerivedFrom: []v1.OriginalTestResults{
				{OriginalFilePath: "path 1"},
				{OriginalFilePath: "path 2"},
			},
			Framework: v1.RubyRSpecFramework,
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
		rubyRSpec1_2 = &v1.TestResults{
			DerivedFrom: []v1.OriginalTestResults{
				{OriginalFilePath: "path 1", GroupNumber: 2},
			},
			Framework: v1.RubyRSpecFramework,
			Summary: v1.Summary{
				Status:     v1.SummaryStatusFailed,
				Tests:      2,
				Successful: 1,
				Failed:     1,
			},
			Tests: []v1.Test{
				{Name: "test 1", Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
				{Name: "test 2", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
			},
		}
		rubyRSpec2 = &v1.TestResults{
			DerivedFrom: []v1.OriginalTestResults{
				{OriginalFilePath: "path 3"},
			},
			Framework: v1.RubyRSpecFramework,
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
	})

	It("unions and flattens results", func() {
		Expect(v1.Merge(
			[]v1.TestResults{*rubyRSpec1, *rubyRSpec2},
			[]v1.TestResults{*rubyRSpec1_2},
		)).To(Equal(
			v1.TestResults{
				DerivedFrom: []v1.OriginalTestResults{
					{OriginalFilePath: "path 1"},
					{OriginalFilePath: "path 2"},
					{OriginalFilePath: "path 3"},
					{OriginalFilePath: "path 1", GroupNumber: 2},
				},
				Framework: v1.RubyRSpecFramework,
				OtherErrors: []v1.OtherError{
					{Message: "other error 1"},
					{Message: "other error 2"},
					{Message: "other error 3"},
				},
				Summary: v1.Summary{
					Status:      v1.SummaryStatusFailed,
					Tests:       6,
					OtherErrors: 3,
					Successful:  3,
					Failed:      1,
					Skipped:     2,
					Retries:     2,
				},
				Tests: []v1.Test{
					{
						Name:         "test 1",
						Attempt:      v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
						PastAttempts: []v1.TestAttempt{{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
					},
					{
						Name:         "test 2",
						Attempt:      v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
						PastAttempts: []v1.TestAttempt{{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
					},
					{Name: "test 3", Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
					{Name: "test 4", Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}},
					{Name: "test 5", Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
					{Name: "test 6", Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}},
				},
			},
		))
	})

	It("flattens on all top-level test fields across many batches", func() {
		str1 := "1"
		str2 := "2"
		str3 := "3"
		str4 := "4"
		str5 := "5"

		int1 := 1
		int2 := 2

		scope1 := "scope1"
		id1 := "id1"
		id1other := "id1"
		name1 := "name1"
		lineage1 := []string{"name", "1"}
		location1 := v1.Location{File: "file1.rb", Line: &int1}
		location1other := v1.Location{File: "file1.rb", Line: &int1}

		scope2 := "scope2"
		id2 := "id2"
		name2 := "name2"
		lineage2 := []string{"name", "2"}
		location2 := v1.Location{File: "file1.rb", Line: &int2}

		results1 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str1, nil, nil)},
				},
				{
					ID:           &id1,
					Name:         name2,
					Lineage:      lineage1,
					Location:     &location1,
					Attempt:      v1.TestAttempt{Status: v1.NewFailedTestStatus(&str2, nil, nil)},
					PastAttempts: []v1.TestAttempt{{Status: v1.NewFailedTestStatus(&str2, nil, nil)}},
				},
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage2,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str3, nil, nil)},
				},
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str4, nil, nil)},
				},
				{
					ID:       &id2,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str5, nil, nil)},
				},
				{
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
				{
					Scope:    &scope1,
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
				{
					Scope:    &scope2,
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
			},
		}

		results2 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1other,
					Name:     name1,
					Lineage:  lineage2,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str3, nil, nil)},
				},
				{
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
				{
					Scope:    &scope1,
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
				},
			},
		}

		results3 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1other,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
				{
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
				},
			},
		}

		Expect(v1.Merge([]v1.TestResults{results1}, []v1.TestResults{results2}, []v1.TestResults{results3})).To(Equal(
			v1.TestResults{
				Framework: v1.RubyRSpecFramework,
				Summary: v1.Summary{
					Status:     v1.SummaryStatusFailed,
					Tests:      8,
					Retries:    5,
					Failed:     4,
					Successful: 4,
				},
				Tests: []v1.Test{
					{
						ID:           &id1,
						Name:         name1,
						Lineage:      lineage1,
						Location:     &location1,
						Attempt:      v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
						PastAttempts: []v1.TestAttempt{{Status: v1.NewFailedTestStatus(&str1, nil, nil)}},
					},
					{
						ID:           &id1,
						Name:         name2,
						Lineage:      lineage1,
						Location:     &location1,
						Attempt:      v1.TestAttempt{Status: v1.NewFailedTestStatus(&str2, nil, nil)},
						PastAttempts: []v1.TestAttempt{{Status: v1.NewFailedTestStatus(&str2, nil, nil)}},
					},
					{
						ID:           &id1,
						Name:         name1,
						Lineage:      lineage2,
						Location:     &location1,
						Attempt:      v1.TestAttempt{Status: v1.NewFailedTestStatus(&str3, nil, nil)},
						PastAttempts: []v1.TestAttempt{{Status: v1.NewFailedTestStatus(&str3, nil, nil)}},
					},
					{
						ID:       &id1,
						Name:     name1,
						Lineage:  lineage1,
						Location: &location2,
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str4, nil, nil)},
					},
					{
						ID:       &id2,
						Name:     name1,
						Lineage:  lineage1,
						Location: &location1,
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str5, nil, nil)},
					},
					{
						ID:       &id2,
						Name:     name2,
						Lineage:  lineage2,
						Location: &location2,
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
						PastAttempts: []v1.TestAttempt{
							{Status: v1.NewSuccessfulTestStatus()},
							{Status: v1.NewCanceledTestStatus()},
						},
					},
					{
						Scope:    &scope1,
						ID:       &id2,
						Name:     name2,
						Lineage:  lineage2,
						Location: &location2,
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
						PastAttempts: []v1.TestAttempt{
							{Status: v1.NewFailedTestStatus(nil, nil, nil)},
						},
					},
					{
						Scope:    &scope2,
						ID:       &id2,
						Name:     name2,
						Lineage:  lineage2,
						Location: &location2,
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
				},
			},
		))
	})

	It("flattens across batches when there are empty batches", func() {
		int2 := 2

		id2 := "id2"
		name2 := "name2"
		lineage2 := []string{"name", "2"}
		location2 := v1.Location{File: "file1.rb", Line: &int2}

		results1 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
			},
		}

		results2 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
			},
		}

		results3 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
			},
		}

		Expect(v1.Merge(
			[]v1.TestResults{results1},
			[]v1.TestResults{},
			[]v1.TestResults{results2},
			[]v1.TestResults{},
			[]v1.TestResults{},
			[]v1.TestResults{results3},
		)).To(Equal(
			v1.TestResults{
				Framework: v1.RubyRSpecFramework,
				Summary: v1.Summary{
					Status:     v1.SummaryStatusSuccessful,
					Tests:      1,
					Retries:    1,
					Successful: 1,
				},
				Tests: []v1.Test{
					{
						ID:       &id2,
						Name:     name2,
						Lineage:  lineage2,
						Location: &location2,
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
						PastAttempts: []v1.TestAttempt{
							{Status: v1.NewSuccessfulTestStatus()},
							{Status: v1.NewSuccessfulTestStatus()},
						},
					},
				},
			},
		))
	})

	It("unions any tests found in batches that were not in previous ones", func() {
		str1 := "1"
		str2 := "2"

		int1 := 1
		int2 := 2

		id1 := "id1"
		name1 := "name1"
		lineage1 := []string{"name", "1"}
		location1 := v1.Location{File: "file1.rb", Line: &int1}

		id2 := "id2"
		name2 := "name2"
		lineage2 := []string{"name", "2"}
		location2 := v1.Location{File: "file1.rb", Line: &int2}

		results1 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str1, nil, nil)},
				},
			},
		}

		results2 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage2,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str2, nil, nil)},
				},
			},
		}

		results3 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id2,
					Name:     name2,
					Lineage:  lineage2,
					Location: &location2,
					Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
				},
			},
		}

		Expect(v1.Merge(
			[]v1.TestResults{results1},
			[]v1.TestResults{results2},
			[]v1.TestResults{results3},
		)).To(Equal(
			v1.TestResults{
				Framework: v1.RubyRSpecFramework,
				Summary: v1.Summary{
					Status:   v1.SummaryStatusFailed,
					Tests:    3,
					Failed:   2,
					Canceled: 1,
				},
				Tests: []v1.Test{
					{
						ID:       &id1,
						Name:     name1,
						Lineage:  lineage1,
						Location: &location1,
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str1, nil, nil)},
					},
					{
						ID:       &id1,
						Name:     name1,
						Lineage:  lineage2,
						Location: &location1,
						Attempt: v1.TestAttempt{
							Status: v1.NewFailedTestStatus(&str2, nil, nil),
							Meta: map[string]any{
								"__rwx": map[string]any{
									"missingInPreviousBatchOfResults": true,
								},
							},
						},
					},
					{
						ID:       &id2,
						Name:     name2,
						Lineage:  lineage2,
						Location: &location2,
						Attempt: v1.TestAttempt{
							Status: v1.NewCanceledTestStatus(),
							Meta: map[string]any{
								"__rwx": map[string]any{
									"missingInPreviousBatchOfResults": true,
								},
							},
						},
					},
				},
			},
		))
	})

	It("only merges incoming tests into one base test, even if there are multiple matches", func() {
		str1 := "1"

		int1 := 1

		id1 := "id1"
		name1 := "name1"
		lineage1 := []string{"name", "1"}
		location1 := v1.Location{File: "file1.rb", Line: &int1}

		results1 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(&str1, nil, nil)},
				},
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
				},
			},
		}

		results2 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
			},
		}

		Expect(v1.Merge(
			[]v1.TestResults{results1},
			[]v1.TestResults{results2},
		)).To(Equal(
			v1.TestResults{
				Framework: v1.RubyRSpecFramework,
				Summary: v1.Summary{
					Status:     v1.SummaryStatusFailed,
					Tests:      2,
					Successful: 1,
					Canceled:   1,
					Retries:    1,
				},
				Tests: []v1.Test{
					{
						ID:       &id1,
						Name:     name1,
						Lineage:  lineage1,
						Location: &location1,
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
						PastAttempts: []v1.TestAttempt{
							{Status: v1.NewFailedTestStatus(&str1, nil, nil)},
						},
					},
					{
						ID:       &id1,
						Name:     name1,
						Lineage:  lineage1,
						Location: &location1,
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
				},
			},
		))
	})

	It("does not flatten attempts that didn't actually run", func() {
		int1 := 1

		id1 := "id1"
		name1 := "name1"
		lineage1 := []string{"name", "1"}
		location1 := v1.Location{File: "file1.rb", Line: &int1}

		results1 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				},
			},
		}

		results2 := v1.TestResults{
			Framework: v1.RubyRSpecFramework,
			Tests: []v1.Test{
				{
					ID:       &id1,
					Name:     name1,
					Lineage:  lineage1,
					Location: &location1,
					Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
				},
			},
		}

		Expect(v1.Merge(
			[]v1.TestResults{results1},
			[]v1.TestResults{results2},
		)).To(Equal(
			v1.TestResults{
				Framework: v1.RubyRSpecFramework,
				Summary: v1.Summary{
					Status:     v1.SummaryStatusSuccessful,
					Tests:      1,
					Successful: 1,
				},
				Tests: []v1.Test{
					{
						ID:       &id1,
						Name:     name1,
						Lineage:  lineage1,
						Location: &location1,
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
				},
			},
		))
	})
})
