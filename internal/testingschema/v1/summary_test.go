package v1_test

import (
	"encoding/json"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Summary", func() {
	Describe("NewSummary", func() {
		It("summarizes no tests and no other errors", func() {
			Expect(v1.NewSummary(nil, nil)).To(Equal(
				v1.Summary{
					Status: v1.SummaryStatusSuccessful,
				},
			))
		})

		It("summarizes test statuses", func() {
			var test v1.Test

			test = v1.Test{Attempt: v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()}}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status:     v1.SummaryStatusSuccessful,
					Tests:      2,
					Successful: 2,
				},
			))

			test = v1.Test{Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)}}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status: v1.SummaryStatusFailed,
					Tests:  2,
					Failed: 2,
				},
			))

			test = v1.Test{Attempt: v1.TestAttempt{Status: v1.NewCanceledTestStatus()}}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status:   v1.SummaryStatusFailed,
					Tests:    2,
					Canceled: 2,
				},
			))

			test = v1.Test{Attempt: v1.TestAttempt{Status: v1.NewPendedTestStatus(nil)}}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status: v1.SummaryStatusSuccessful,
					Tests:  2,
					Pended: 2,
				},
			))

			test = v1.Test{Attempt: v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)}}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status:  v1.SummaryStatusSuccessful,
					Tests:   2,
					Skipped: 2,
				},
			))

			test = v1.Test{Attempt: v1.TestAttempt{Status: v1.NewTimedOutTestStatus()}}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status:   v1.SummaryStatusFailed,
					Tests:    2,
					TimedOut: 2,
				},
			))

			test = v1.Test{Attempt: v1.TestAttempt{Status: v1.NewTodoTestStatus(nil)}}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status: v1.SummaryStatusSuccessful,
					Tests:  2,
					Todo:   2,
				},
			))

			test = v1.Test{Attempt: v1.TestAttempt{
				Status: v1.NewQuarantinedTestStatus(v1.NewFailedTestStatus(nil, nil, nil)),
			}}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status:      v1.SummaryStatusSuccessful,
					Tests:       2,
					Quarantined: 2,
				},
			))
		})

		It("summarizes other errors", func() {
			Expect(v1.NewSummary(nil, []v1.OtherError{{Message: "other error"}, {Message: "other error 2"}})).To(Equal(
				v1.Summary{
					Status:      v1.SummaryStatusFailed,
					OtherErrors: 2,
				},
			))
		})

		It("summarizes retries", func() {
			test := v1.Test{
				Attempt: v1.TestAttempt{
					Status: v1.NewSuccessfulTestStatus(),
				},
				PastAttempts: []v1.TestAttempt{
					{
						Status: v1.NewFailedTestStatus(nil, nil, nil),
					},
					{
						Status: v1.NewFailedTestStatus(nil, nil, nil),
					},
				},
			}
			Expect(v1.NewSummary([]v1.Test{test, test}, nil)).To(Equal(
				v1.Summary{
					Status:     v1.SummaryStatusSuccessful,
					Tests:      2,
					Retries:    2,
					Successful: 2,
				},
			))
		})
	})

	Describe("SummaryStatus", func() {
		It("marshals the summary statuses", func() {
			var data []byte
			var err error

			data, err = json.Marshal(v1.SummaryStatusSuccessful)

			Expect(err).To(BeNil())
			Expect(string(data)).To(Equal(`{"kind":"successful"}`))

			data, err = json.Marshal(v1.SummaryStatusFailed)

			Expect(err).To(BeNil())
			Expect(string(data)).To(Equal(`{"kind":"failed"}`))

			data, err = json.Marshal(v1.SummaryStatusCanceled)

			Expect(err).To(BeNil())
			Expect(string(data)).To(Equal(`{"kind":"canceled"}`))

			data, err = json.Marshal(v1.SummaryStatusTimedOut)

			Expect(err).To(BeNil())
			Expect(string(data)).To(Equal(`{"kind":"timedOut"}`))

			data, err = json.Marshal(v1.SummaryStatus("foo"))

			Expect(err).To(BeNil())
			Expect(string(data)).To(Equal(`{"kind":"foo"}`))
		})

		It("unmarshals the summary statuses", func() {
			var ss v1.SummaryStatus
			var err error
			var data string

			data = `{"kind":"successful"}`
			err = json.Unmarshal([]byte(data), &ss)

			Expect(err).To(BeNil())
			Expect(ss).To(Equal(v1.SummaryStatusSuccessful))

			data = `{"kind":"failed"}`
			err = json.Unmarshal([]byte(data), &ss)

			Expect(err).To(BeNil())
			Expect(ss).To(Equal(v1.SummaryStatusFailed))

			data = `{"kind":"canceled"}`
			err = json.Unmarshal([]byte(data), &ss)

			Expect(err).To(BeNil())
			Expect(ss).To(Equal(v1.SummaryStatusCanceled))

			data = `{"kind":"timedOut"}`
			err = json.Unmarshal([]byte(data), &ss)

			Expect(err).To(BeNil())
			Expect(ss).To(Equal(v1.SummaryStatusTimedOut))

			data = `{"kind":"foo"}`
			err = json.Unmarshal([]byte(data), &ss)

			Expect(err).To(BeNil())
			Expect(ss).To(Equal(v1.SummaryStatus("foo")))
		})
	})
})
