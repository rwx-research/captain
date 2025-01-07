package testresultsschema

import (
	"encoding/json"

	"github.com/rwx-research/captain-cli/internal/errors"
)

type SummaryStatus string

const (
	SummaryStatusSuccessful SummaryStatus = "successful"
	SummaryStatusCanceled   SummaryStatus = "canceled"
	SummaryStatusFailed     SummaryStatus = "failed"
	SummaryStatusTimedOut   SummaryStatus = "timedOut"
)

type summaryStatus struct {
	Kind string `json:"kind"`
}

func (ss SummaryStatus) MarshalJSON() ([]byte, error) {
	json, err := json.Marshal(&summaryStatus{Kind: (string)(ss)})
	return json, errors.WithStack(err)
}

func (ss *SummaryStatus) UnmarshalJSON(b []byte) error {
	var s summaryStatus
	if err := json.Unmarshal(b, &s); err != nil {
		return errors.WithStack(err)
	}

	*ss = SummaryStatus(s.Kind)
	return nil
}

type Summary struct {
	Status      SummaryStatus `json:"status"`
	Tests       int           `json:"tests"`
	OtherErrors int           `json:"otherErrors"`
	Retries     int           `json:"retries"`
	Canceled    int           `json:"canceled"`
	Failed      int           `json:"failed"`
	Pended      int           `json:"pended"`
	Quarantined int           `json:"quarantined"`
	Skipped     int           `json:"skipped"`
	Successful  int           `json:"successful"`
	TimedOut    int           `json:"timedOut"`
	Todo        int           `json:"todo"`
}

func NewSummary(tests []Test, otherErrors []OtherError) Summary {
	summary := Summary{Tests: len(tests), OtherErrors: len(otherErrors)}
	status := SummaryStatusSuccessful

	if len(otherErrors) > 0 {
		status = SummaryStatusFailed
	}

	for _, test := range tests {
		if len(test.PastAttempts) > 0 {
			summary.Retries++
		}

		if test.Attempt.Status.ImpliesFailure() {
			status = SummaryStatusFailed
		}

		if test.Attempt.Status.Kind == TestStatusCanceled {
			summary.Canceled++
		}
		if test.Attempt.Status.Kind == TestStatusFailed {
			summary.Failed++
		}
		if test.Attempt.Status.Kind == TestStatusPended {
			summary.Pended++
		}
		if test.Attempt.Status.Kind == TestStatusQuarantined {
			summary.Quarantined++
		}
		if test.Attempt.Status.Kind == TestStatusSkipped {
			summary.Skipped++
		}
		if test.Attempt.Status.Kind == TestStatusSuccessful {
			summary.Successful++
		}
		if test.Attempt.Status.Kind == TestStatusTimedOut {
			summary.TimedOut++
		}
		if test.Attempt.Status.Kind == TestStatusTodo {
			summary.Todo++
		}
	}

	summary.Status = status
	return summary
}
