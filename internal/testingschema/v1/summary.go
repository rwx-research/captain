package v1

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
	return json, errors.Wrap(err)
}

func (ss *SummaryStatus) UnmarshalJSON(b []byte) error {
	var s summaryStatus
	if err := json.Unmarshal(b, &s); err != nil {
		println(err)
		return errors.Wrap(err)
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
