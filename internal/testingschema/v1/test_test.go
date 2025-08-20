package v1_test

import (
	"encoding/json"
	"time"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test", func() {
	Describe("TestAttempt", func() {
		It("serializes the duration to nanoseconds", func() {
			duration := time.Duration(1500000000)
			json, err := json.Marshal(v1.TestAttempt{Duration: &duration})

			Expect(err).To(BeNil())
			Expect(string(json)).To(ContainSubstring(`"durationInNanoseconds":1500000000`))
		})

		It("serializes no duration to null", func() {
			json, err := json.Marshal(v1.TestAttempt{Duration: nil})

			Expect(err).To(BeNil())
			Expect(string(json)).To(ContainSubstring(`"durationInNanoseconds":null`))
		})
	})

	Describe("NewCanceledTestStatus", func() {
		It("produces a Canceled test status", func() {
			Expect(v1.NewCanceledTestStatus()).To(Equal(
				v1.TestStatus{
					Kind: v1.TestStatusCanceled,
				},
			))
		})
	})

	Describe("NewFailedTestStatus", func() {
		It("produces a Failed test status", func() {
			message := "message"
			exception := "exception"
			backtrace := []string{"1", "2", "3"}
			Expect(v1.NewFailedTestStatus(&message, &exception, backtrace)).To(Equal(
				v1.TestStatus{
					Kind:      v1.TestStatusFailed,
					Message:   &message,
					Exception: &exception,
					Backtrace: backtrace,
				},
			))
		})
	})

	Describe("NewPendedTestStatus", func() {
		It("produces a Pended test status", func() {
			message := "message"
			Expect(v1.NewPendedTestStatus(&message)).To(Equal(
				v1.TestStatus{
					Kind:    v1.TestStatusPended,
					Message: &message,
				},
			))
		})
	})

	Describe("NewSkippedTestStatus", func() {
		It("produces a Skipped test status", func() {
			message := "message"
			Expect(v1.NewSkippedTestStatus(&message)).To(Equal(
				v1.TestStatus{
					Kind:    v1.TestStatusSkipped,
					Message: &message,
				},
			))
		})
	})

	Describe("NewSuccessfulTestStatus", func() {
		It("produces a Successful test status", func() {
			Expect(v1.NewSuccessfulTestStatus()).To(Equal(
				v1.TestStatus{
					Kind: v1.TestStatusSuccessful,
				},
			))
		})
	})

	Describe("NewTimedOutTestStatus", func() {
		It("produces a TimedOut test status", func() {
			Expect(v1.NewTimedOutTestStatus(nil, nil, nil)).To(Equal(
				v1.TestStatus{
					Kind: v1.TestStatusTimedOut,
				},
			))
		})
	})

	Describe("NewTodoTestStatus", func() {
		It("produces a Todo test status", func() {
			message := "message"
			Expect(v1.NewTodoTestStatus(&message)).To(Equal(
				v1.TestStatus{
					Kind:    v1.TestStatusTodo,
					Message: &message,
				},
			))
		})
	})

	Describe("NewQuarantinedTestStatus", func() {
		It("produces a Quarantined test status", func() {
			originalStatus := v1.NewCanceledTestStatus()
			Expect(v1.NewQuarantinedTestStatus(originalStatus)).To(Equal(
				v1.TestStatus{
					Kind:           v1.TestStatusQuarantined,
					OriginalStatus: &originalStatus,
				},
			))
		})
	})

	Describe("ImpliesSkipped", func() {
		It("implies skipped for failed statuses", func() {
			Expect(v1.NewPendedTestStatus(nil).ImpliesSkipped()).To(Equal(true))
		})

		It("implies skipped for canceled statuses", func() {
			Expect(v1.NewTodoTestStatus(nil).ImpliesSkipped()).To(Equal(true))
		})

		It("implies skipped for timed out statuses", func() {
			Expect(v1.NewSkippedTestStatus(nil).ImpliesSkipped()).To(Equal(true))
		})

		It("does not imply skipped for other statuses", func() {
			Expect(v1.NewSuccessfulTestStatus().ImpliesSkipped()).To(Equal(false))
		})
	})

	Describe("ImpliesFailure", func() {
		It("implies failure for failed statuses", func() {
			Expect(v1.NewFailedTestStatus(nil, nil, nil).ImpliesFailure()).To(Equal(true))
		})

		It("implies failure for canceled statuses", func() {
			Expect(v1.NewCanceledTestStatus().ImpliesFailure()).To(Equal(true))
		})

		It("implies failure for timed out statuses", func() {
			Expect(v1.NewTimedOutTestStatus(nil, nil, nil).ImpliesFailure()).To(Equal(true))
		})

		It("does not imply failure for other statuses", func() {
			Expect(v1.NewSuccessfulTestStatus().ImpliesFailure()).To(Equal(false))
		})
	})

	Describe("PotentiallyFlaky", func() {
		It("is potentially flaky for failed statuses", func() {
			Expect(v1.NewFailedTestStatus(nil, nil, nil).PotentiallyFlaky()).To(Equal(true))
		})

		It("is not potentially flaky for canceled statuses", func() {
			Expect(v1.NewCanceledTestStatus().PotentiallyFlaky()).To(Equal(false))
		})

		It("is potentially flaky for timed out statuses", func() {
			Expect(v1.NewTimedOutTestStatus(nil, nil, nil).PotentiallyFlaky()).To(Equal(true))
		})

		It("is not potentially flaky for other statuses", func() {
			Expect(v1.NewSuccessfulTestStatus().PotentiallyFlaky()).To(Equal(false))
		})
	})

	Describe("Flaky", func() {
		It("is false for a test that passed", func() {
			test := v1.Test{Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}}
			Expect(test.Flaky()).To(Equal(false))
		})

		It("is false for a test that failed", func() {
			test := v1.Test{Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}}
			Expect(test.Flaky()).To(Equal(false))
		})

		It("is false for a test that timed out", func() {
			test := v1.Test{Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)}}
			Expect(test.Flaky()).To(Equal(false))
		})

		It("is false for a test that was canceled", func() {
			test := v1.Test{Attempt: v1.TestAttempt{Status: v1.NewCanceledTestStatus()}}
			Expect(test.Flaky()).To(Equal(false))
		})

		It("is true for a test that failed then passed", func() {
			test := v1.Test{
				Attempt:      v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
				PastAttempts: []v1.TestAttempt{{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
			}
			Expect(test.Flaky()).To(Equal(true))
		})

		It("is true for a test that passed then failed", func() {
			test := v1.Test{
				Attempt:      v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
				PastAttempts: []v1.TestAttempt{{Status: v1.NewSuccessfulTestStatus()}},
			}
			Expect(test.Flaky()).To(Equal(true))
		})

		It("is true for a test that timed out and passed", func() {
			test := v1.Test{
				Attempt:      v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)},
				PastAttempts: []v1.TestAttempt{{Status: v1.NewSuccessfulTestStatus()}},
			}
			Expect(test.Flaky()).To(Equal(true))
		})

		It("is false for a test that was canceled and passed", func() {
			test := v1.Test{
				Attempt:      v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
				PastAttempts: []v1.TestAttempt{{Status: v1.NewSuccessfulTestStatus()}},
			}
			Expect(test.Flaky()).To(Equal(false))
		})

		It("is false for a test that was pended and failed", func() {
			test := v1.Test{
				Attempt:      v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
				PastAttempts: []v1.TestAttempt{{Status: v1.NewFailedTestStatus(nil, nil, nil)}},
			}
			Expect(test.Flaky()).To(Equal(false))
		})

		It("is false for a test that was pended and passed", func() {
			test := v1.Test{
				Attempt:      v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)},
				PastAttempts: []v1.TestAttempt{{Status: v1.NewSuccessfulTestStatus()}},
			}
			Expect(test.Flaky()).To(Equal(false))
		})
	})

	Describe("Quarantine", func() {
		It("quarantines a test that is not quarantined", func() {
			originalStatus := v1.NewFailedTestStatus(nil, nil, nil)
			test := v1.Test{Attempt: v1.TestAttempt{Status: originalStatus}}
			Expect(test.Attempt.Status).To(Equal(originalStatus))

			quarantinedTest := test.Quarantine()
			Expect(test.Attempt.Status).To(Equal(originalStatus))
			Expect(quarantinedTest.Attempt.Status).To(Equal(v1.NewQuarantinedTestStatus(originalStatus)))
		})

		It("does not double-quarantine a test", func() {
			originalStatus := v1.NewFailedTestStatus(nil, nil, nil)
			test := v1.Test{Attempt: v1.TestAttempt{Status: v1.NewQuarantinedTestStatus(originalStatus)}}
			Expect(test.Attempt.Status).To(Equal(v1.NewQuarantinedTestStatus(originalStatus)))

			quarantinedTest := test.Quarantine()
			Expect(test.Attempt.Status).To(Equal(v1.NewQuarantinedTestStatus(originalStatus)))
			Expect(quarantinedTest).To(Equal(test))
		})
	})

	Describe("Tag", func() {
		It("adds RWX metadata to the test when there is no existing meta", func() {
			Expect(v1.Test{Attempt: v1.TestAttempt{}}.Tag("some-key", true)).To(Equal(
				v1.Test{
					Attempt: v1.TestAttempt{
						Meta: map[string]any{
							"__rwx": map[string]any{"some-key": true},
						},
					},
				},
			))
		})

		It("adds RWX metadata to the test when there is existing meta, but not RWX meta", func() {
			Expect(
				v1.Test{
					Attempt: v1.TestAttempt{Meta: map[string]any{"foo": "bar"}},
				}.Tag("some-key", true),
			).To(Equal(
				v1.Test{
					Attempt: v1.TestAttempt{
						Meta: map[string]any{
							"foo":   "bar",
							"__rwx": map[string]any{"some-key": true},
						},
					},
				},
			))
		})

		It("adds RWX metadata to the test when there is existing RWX meta", func() {
			Expect(
				v1.Test{
					Attempt: v1.TestAttempt{Meta: map[string]any{
						"foo":   "bar",
						"__rwx": map[string]any{"something": "else"},
					}},
				}.Tag("some-key", true),
			).To(Equal(
				v1.Test{
					Attempt: v1.TestAttempt{
						Meta: map[string]any{
							"foo": "bar",
							"__rwx": map[string]any{
								"something": "else",
								"some-key":  true,
							},
						},
					},
				},
			))
		})

		It("does not add RWX metadata to the test when there is existing RWX meta in an unexpected shape", func() {
			Expect(
				v1.Test{
					Attempt: v1.TestAttempt{Meta: map[string]any{
						"foo":   "bar",
						"__rwx": true,
					}},
				}.Tag("some-key", true),
			).To(Equal(
				v1.Test{
					Attempt: v1.TestAttempt{
						Meta: map[string]any{
							"foo":   "bar",
							"__rwx": true,
						},
					},
				},
			))
		})
	})

	Describe("Matches", func() {
		It("matches when the top-level fields are the same", func() {
			scope1_1 := "scope1"
			id1_1 := "id1"
			name1_1 := "name1"
			lineage1_1 := []string{"name", "1"}
			file1_1 := "file1"
			location1_1 := v1.Location{File: file1_1}

			scope1_2 := "scope1"
			id1_2 := "id1"
			name1_2 := "name1"
			lineage1_2 := []string{"name", "1"}
			file1_2 := "file1"
			location1_2 := v1.Location{File: file1_2}

			test := v1.Test{
				Scope:    &scope1_1,
				ID:       &id1_1,
				Name:     name1_1,
				Lineage:  lineage1_1,
				Location: &location1_1,
			}

			Expect(test.Matches(v1.Test{
				Scope:    &scope1_2,
				ID:       &id1_2,
				Name:     name1_2,
				Lineage:  lineage1_2,
				Location: &location1_2,
			})).To(BeTrue())

			Expect(test.Matches(v1.Test{
				Scope:    nil,
				ID:       &id1_2,
				Name:     name1_2,
				Lineage:  lineage1_2,
				Location: &location1_2,
			})).To(BeFalse())

			Expect(test.Matches(v1.Test{
				Scope:    &scope1_2,
				ID:       nil,
				Name:     name1_2,
				Lineage:  lineage1_2,
				Location: &location1_2,
			})).To(BeFalse())

			Expect(test.Matches(v1.Test{
				Scope:    &scope1_2,
				ID:       &id1_2,
				Name:     name1_2,
				Lineage:  lineage1_2,
				Location: nil,
			})).To(BeFalse())

			Expect(test.Matches(v1.Test{
				Scope:    &scope1_2,
				ID:       &id1_2,
				Name:     "other name",
				Lineage:  lineage1_2,
				Location: &location1_2,
			})).To(BeFalse())

			Expect(test.Matches(v1.Test{
				Scope:    &scope1_2,
				ID:       &id1_2,
				Name:     name1_2,
				Lineage:  []string{"other"},
				Location: &location1_2,
			})).To(BeFalse())
		})
	})

	Describe("IdentityForMatching", func() {
		It("constructs a string based on all of the attributes used for matching", func() {
			scope1_1 := "scope1"
			id1_1 := "id1"
			name1_1 := "name1"
			lineage1_1 := []string{"name", "1"}
			file1_1 := "file1"
			column := 1
			line := 2
			location1_1 := v1.Location{File: file1_1, Column: &column, Line: &line}

			test := v1.Test{
				Scope:    &scope1_1,
				ID:       &id1_1,
				Name:     name1_1,
				Lineage:  lineage1_1,
				Location: &location1_1,
			}

			//nolint:lll
			Expect(test.IdentityForMatching()).To(Equal("scope=scope1 :: id=id1 :: name=name1 :: locationFile=file1 :: locationColumn=1 :: locationLine=2 :: lineage=____name____1"))
		})

		It("uses 'nil' for some nil values and empty string for others", func() {
			name1_1 := "name1"
			lineage1_1 := []string{}

			test := v1.Test{
				Name:    name1_1,
				Lineage: lineage1_1,
			}

			//nolint:lll
			Expect(test.IdentityForMatching()).To(Equal("scope= :: id=nil :: name=name1 :: locationFile=nil :: locationColumn=nil :: locationLine=nil :: lineage="))
		})
	})

	Describe("Identify", func() {
		Context("with strict identification", func() {
			It("returns an error when fetching from meta of a test without meta", func() {
				compositeIdentifier, err := v1.Test{
					Name:    "the-description",
					Attempt: v1.TestAttempt{},
				}.Identify(v1.TestIdentityRecipe{Components: []string{"description", "something_from_meta"}, Strict: true})

				Expect(compositeIdentifier).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("Meta is not defined"))
			})

			It("returns an error when fetching from meta of a test without the component in meta", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
					Attempt: v1.TestAttempt{
						Meta: map[string]any{"something_else_in_meta": 1},
					},
				}.Identify(v1.TestIdentityRecipe{Components: []string{"description", "something_from_meta"}, Strict: true})

				Expect(compositeIdentifier).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("it was not there"))
			})

			It("returns an error when the fetching the file without a location", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
				}.Identify(v1.TestIdentityRecipe{Components: []string{"description", "file"}, Strict: true})

				Expect(compositeIdentifier).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("Location is not defined"))
			})

			It("returns an error when the fetching the ID when it's not there", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
				}.Identify(v1.TestIdentityRecipe{Components: []string{"description", "id"}, Strict: true})

				Expect(compositeIdentifier).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("ID is not defined"))
			})

			It("returns a composite identifier otherwise", func() {
				id := "the-id"
				compositeIdentifier, err := v1.Test{
					ID:       &id,
					Name:     "the-description",
					Location: &v1.Location{File: "the-file"},
					Attempt: v1.TestAttempt{
						Meta: map[string]any{
							"foo": 1,
							"bar": "hello",
							"baz": false,
							"nil": nil,
						},
					},
				}.Identify(v1.TestIdentityRecipe{
					Components: []string{"id", "description", "file", "foo", "bar", "baz", "nil"},
					Strict:     true,
				})

				Expect(compositeIdentifier).To(Equal(
					"the-id -captain- the-description -captain- the-file -captain-" +
						" 1 -captain- hello -captain- false -captain- ",
				))
				Expect(err).To(BeNil())
			})
		})

		Context("without strict identification", func() {
			It("returns a composite identifier when fetching from meta of a test without meta", func() {
				compositeIdentifier, err := v1.Test{
					Name:    "the-description",
					Attempt: v1.TestAttempt{},
				}.Identify(v1.TestIdentityRecipe{Components: []string{"description", "something_from_meta"}, Strict: false})

				Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
				Expect(err).To(BeNil())
			})

			It("returns a composite identifier when fetching from meta of a test without the component in meta", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
					Attempt: v1.TestAttempt{
						Meta: map[string]any{"something_else_in_meta": 1},
					},
				}.Identify(v1.TestIdentityRecipe{Components: []string{"description", "something_from_meta"}, Strict: false})

				Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
				Expect(err).To(BeNil())
			})

			It("returns a composite identifier when fetching the file without a location", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
				}.Identify(v1.TestIdentityRecipe{Components: []string{"description", "file"}, Strict: false})

				Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
				Expect(err).To(BeNil())
			})

			It("returns a composite identifier when fetching the ID when it's not there", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
				}.Identify(v1.TestIdentityRecipe{Components: []string{"description", "id"}, Strict: false})

				Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
				Expect(err).To(BeNil())
			})

			It("returns a composite identifier otherwise", func() {
				id := "the-id"
				compositeIdentifier, err := v1.Test{
					ID:       &id,
					Name:     "the-description",
					Location: &v1.Location{File: "the-file"},
					Attempt: v1.TestAttempt{
						Meta: map[string]any{
							"foo": 1,
							"bar": "hello",
							"baz": false,
							"nil": nil,
						},
					},
				}.Identify(v1.TestIdentityRecipe{
					Components: []string{"id", "description", "file", "foo", "bar", "baz", "nil"},
					Strict:     false,
				})

				Expect(compositeIdentifier).To(Equal(
					"the-id -captain- the-description -captain- the-file -captain-" +
						" 1 -captain- hello -captain- false -captain- ",
				))
				Expect(err).To(BeNil())
			})
		})
	})
})
