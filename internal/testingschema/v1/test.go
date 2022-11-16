package v1

import "time"

type TestStatus struct {
	Kind           string      `json:"kind,omitempty"`
	OriginalStatus *TestStatus `json:"originalStatus,omitempty"`
	Message        *string     `json:"message,omitempty"`
	Exception      *string     `json:"exception,omitempty"`
	Backtrace      []string    `json:"backtrace,omitempty"`
}

type TestAttempt struct {
	Duration   *time.Duration `json:"durationInNanoseconds"`
	Meta       map[string]any `json:"meta,omitempty"`
	Status     TestStatus     `json:"status"`
	Stderr     *string        `json:"stderr,omitempty"`
	Stdout     *string        `json:"stdout,omitempty"`
	StartedAt  *time.Time     `json:"startedAt,omitempty"`
	FinishedAt *time.Time     `json:"finishedAt,omitempty"`
}

type Test struct {
	ID           *string       `json:"id,omitempty"`
	Name         string        `json:"name"`
	Lineage      []string      `json:"lineage,omitempty"`
	Location     *Location     `json:"location,omitempty"`
	Attempt      TestAttempt   `json:"attempt"`
	PastAttempts []TestAttempt `json:"pastAttempts,omitempty"`
}
