package v1

import "time"

type TestStatusKind string

const (
	TestStatusCanceled    TestStatusKind = "canceled"
	TestStatusFailed      TestStatusKind = "failed"
	TestStatusPended      TestStatusKind = "pended"
	TestStatusSkipped     TestStatusKind = "skipped"
	TestStatusSuccessful  TestStatusKind = "successful"
	TestStatusTimedOut    TestStatusKind = "timedOut"
	TestStatusTodo        TestStatusKind = "todo"
	TestStatusQuarantined TestStatusKind = "quarantined"
)

type TestStatus struct {
	Kind           TestStatusKind `json:"kind"`
	OriginalStatus *TestStatus    `json:"originalStatus,omitempty"`
	Message        *string        `json:"message,omitempty"`
	Exception      *string        `json:"exception,omitempty"`
	Backtrace      []string       `json:"backtrace,omitempty"`
}

func NewCanceledTestStatus() TestStatus {
	return TestStatus{Kind: TestStatusCanceled}
}

func NewFailedTestStatus(message *string, exception *string, backtrace []string) TestStatus {
	return TestStatus{
		Kind:      TestStatusFailed,
		Message:   message,
		Exception: exception,
		Backtrace: backtrace,
	}
}

func NewPendedTestStatus(message *string) TestStatus {
	return TestStatus{Kind: TestStatusPended, Message: message}
}

func NewSkippedTestStatus(message *string) TestStatus {
	return TestStatus{Kind: TestStatusSkipped, Message: message}
}

func NewSuccessfulTestStatus() TestStatus {
	return TestStatus{Kind: TestStatusSuccessful}
}

func NewTimedOutTestStatus() TestStatus {
	return TestStatus{Kind: TestStatusTimedOut}
}

func NewTodoTestStatus(message *string) TestStatus {
	return TestStatus{Kind: TestStatusTodo, Message: message}
}

func NewQuarantinedTestStatus(originalStatus TestStatus) TestStatus {
	return TestStatus{Kind: TestStatusQuarantined, OriginalStatus: &originalStatus}
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
