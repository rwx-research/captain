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

	Describe("Quarantine", func() {
		It("quarantines a test that is not quarantined", func() {
			originalStatus := v1.NewFailedTestStatus(nil, nil, nil)
			test := v1.Test{Attempt: v1.TestAttempt{Status: originalStatus}}
			Expect(test.Attempt.Status).To(Equal(originalStatus))

			test.Quarantine()
			Expect(test.Attempt.Status).To(Equal(v1.NewQuarantinedTestStatus(originalStatus)))
		})

		It("does not double-quarantine a test", func() {
			originalStatus := v1.NewFailedTestStatus(nil, nil, nil)
			test := v1.Test{Attempt: v1.TestAttempt{Status: v1.NewQuarantinedTestStatus(originalStatus)}}
			Expect(test.Attempt.Status).To(Equal(v1.NewQuarantinedTestStatus(originalStatus)))

			test.Quarantine()
			Expect(test.Attempt.Status).To(Equal(v1.NewQuarantinedTestStatus(originalStatus)))
		})
	})
})
