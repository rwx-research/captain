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
			Expect(v1.NewTimedOutTestStatus()).To(Equal(
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

	Describe("ImpliesFailure", func() {
		It("implies failure for failed statuses", func() {
			Expect(v1.NewFailedTestStatus(nil, nil, nil).ImpliesFailure()).To(Equal(true))
		})

		It("implies failure for canceled statuses", func() {
			Expect(v1.NewCanceledTestStatus().ImpliesFailure()).To(Equal(true))
		})

		It("implies failure for timed out statuses", func() {
			Expect(v1.NewTimedOutTestStatus().ImpliesFailure()).To(Equal(true))
		})

		It("does not imply failure for other statuses", func() {
			Expect(v1.NewSuccessfulTestStatus().ImpliesFailure()).To(Equal(false))
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

	Describe("Identify", func() {
		Context("with strict identification", func() {
			It("returns an error when fetching from meta of a test without meta", func() {
				compositeIdentifier, err := v1.Test{
					Name:    "the-description",
					Attempt: v1.TestAttempt{},
				}.Identify([]string{"description", "something_from_meta"}, true)

				Expect(compositeIdentifier).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("Meta is not defined"))
			})

			It("returns an error when fetching from meta of a test without the component in meta", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
					Attempt: v1.TestAttempt{
						Meta: map[string]any{"something_else_in_meta": 1},
					},
				}.Identify([]string{"description", "something_from_meta"}, true)

				Expect(compositeIdentifier).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("it was not there"))
			})

			It("returns an error when the fetching the file without a location", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
				}.Identify([]string{"description", "file"}, true)

				Expect(compositeIdentifier).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("Location is not defined"))
			})

			It("returns an error when the fetching the ID when it's not there", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
				}.Identify([]string{"description", "id"}, true)

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
				}.Identify([]string{"id", "description", "file", "foo", "bar", "baz", "nil"}, true)

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
				}.Identify([]string{"description", "something_from_meta"}, false)

				Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
				Expect(err).To(BeNil())
			})

			It("returns a composite identifier when fetching from meta of a test without the component in meta", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
					Attempt: v1.TestAttempt{
						Meta: map[string]any{"something_else_in_meta": 1},
					},
				}.Identify([]string{"description", "something_from_meta"}, false)

				Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
				Expect(err).To(BeNil())
			})

			It("returns a composite identifier when fetching the file without a location", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
				}.Identify([]string{"description", "file"}, false)

				Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
				Expect(err).To(BeNil())
			})

			It("returns a composite identifier when fetching the ID when it's not there", func() {
				compositeIdentifier, err := v1.Test{
					Name: "the-description",
				}.Identify([]string{"description", "id"}, false)

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
				}.Identify([]string{"id", "description", "file", "foo", "bar", "baz", "nil"}, false)

				Expect(compositeIdentifier).To(Equal(
					"the-id -captain- the-description -captain- the-file -captain-" +
						" 1 -captain- hello -captain- false -captain- ",
				))
				Expect(err).To(BeNil())
			})
		})
	})
})
