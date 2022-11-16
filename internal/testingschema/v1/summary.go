package v1

type SummaryStatus struct {
	Kind string `json:"kind"`
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
