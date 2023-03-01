package v1

import (
	"fmt"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
)

type TestStatusKind string

const (
	// successful
	TestStatusSuccessful TestStatusKind = "successful"

	// failed, but ignored
	TestStatusQuarantined TestStatusKind = "quarantined"

	// failures
	TestStatusCanceled TestStatusKind = "canceled"
	TestStatusFailed   TestStatusKind = "failed"
	TestStatusTimedOut TestStatusKind = "timedOut"

	// skipped
	TestStatusPended  TestStatusKind = "pended"
	TestStatusSkipped TestStatusKind = "skipped"
	TestStatusTodo    TestStatusKind = "todo"
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

func (s TestStatus) ImpliesSkipped() bool {
	return s.Kind == TestStatusPended || s.Kind == TestStatusSkipped || s.Kind == TestStatusTodo
}

func (s TestStatus) ImpliesFailure() bool {
	return s.Kind == TestStatusFailed || s.Kind == TestStatusCanceled || s.Kind == TestStatusTimedOut
}

func (s TestStatus) PotentiallyFlaky() bool {
	return s.Kind == TestStatusFailed || s.Kind == TestStatusTimedOut
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
	// Used to disambiguate tests during the merge process, this should align
	// with any fields that are used for test identity, but are not top-level
	// fields (i.e. any identity components that come from the attempt meta).
	// Not included in the JSON output nor the schema.
	Scope *string `json:"-"`

	ID           *string       `json:"id,omitempty"`
	Name         string        `json:"name"`
	Lineage      []string      `json:"lineage,omitempty"`
	Location     *Location     `json:"location,omitempty"`
	Attempt      TestAttempt   `json:"attempt"`
	PastAttempts []TestAttempt `json:"pastAttempts,omitempty"`
}

func (t Test) Quarantine() Test {
	if t.Attempt.Status.Kind == TestStatusQuarantined {
		return t
	}

	t.Attempt.Status = NewQuarantinedTestStatus(t.Attempt.Status)
	return t
}

func (t Test) Tag(key string, value any) Test {
	if t.Attempt.Meta == nil {
		t.Attempt.Meta = map[string]any{}
	}

	if t.Attempt.Meta["__rwx"] == nil {
		t.Attempt.Meta["__rwx"] = map[string]any{key: value}
	} else {
		if rwxMeta, ok := t.Attempt.Meta["__rwx"].(map[string]any); ok {
			rwxMeta[key] = value
			t.Attempt.Meta["__rwx"] = rwxMeta
		}
	}

	return t
}

func (t Test) Matches(other Test) bool {
	if !stringPointerEquals(t.Scope, other.Scope) {
		return false
	}
	if !stringPointerEquals(t.ID, other.ID) {
		return false
	}
	if t.Name != other.Name {
		return false
	}
	if !locationPointerEquals(t.Location, other.Location) {
		return false
	}
	if len(t.Lineage) != len(other.Lineage) {
		return false
	}
	lineageMatches := true
	for i, component := range t.Lineage {
		if other.Lineage[i] != component {
			lineageMatches = false
			break
		}
	}
	return lineageMatches
}

// Calculates the composite identifier of a Test given the components which determine it
func (t Test) Identify(withComponents []string, strictly bool) (string, error) {
	foundComponents := make([]string, 0)

	for _, component := range withComponents {
		var getter func() (*string, error)
		switch component {
		case "description":
			getter = t.nameGetter
		case "file":
			getter = t.fileGetter
		case "id":
			getter = t.idGetter
		default:
			getter = t.metaGetter(component)
		}

		component, err := t.componentValue(strictly, getter)
		if err != nil {
			return "", err
		}
		foundComponents = append(foundComponents, *component)
	}

	return strings.Join(foundComponents, " -captain- "), nil
}

func (t Test) componentValue(strictly bool, getter func() (*string, error)) (*string, error) {
	value, err := getter()

	switch {
	case strictly && err != nil:
		return nil, err
	case err == nil && value == nil:
		zero := ""
		return &zero, nil
	case err == nil && value != nil:
		return value, nil
	default:
		missing := "MISSING_IDENTITY_COMPONENT"
		return &missing, nil
	}
}

func (t Test) nameGetter() (*string, error) {
	return &t.Name, nil
}

func (t Test) fileGetter() (*string, error) {
	if t.Location == nil {
		return nil, errors.NewInternalError(
			"Location is not defined for %v, but we tried to use it for identification.",
			t,
		)
	}

	return &t.Location.File, nil
}

func (t Test) idGetter() (*string, error) {
	if t.ID == nil {
		return nil, errors.NewInternalError(
			"ID is not defined for %v, but we tried to use it for identification.",
			t,
		)
	}

	return t.ID, nil
}

func (t Test) metaGetter(component string) func() (*string, error) {
	return func() (*string, error) {
		if t.Attempt.Meta == nil {
			return nil, errors.NewInternalError(
				"Meta is not defined for %v, but we tried to get %s from it.",
				t,
				component,
			)
		}

		value, exists := t.Attempt.Meta[component]
		if !exists {
			return nil, errors.NewInternalError(
				"Tried to get %s from %v of %v, but it was not there.",
				component,
				t.Attempt.Meta,
				t,
			)
		}

		if value == nil {
			return nil, nil
		}

		formatted := fmt.Sprintf("%v", value)
		return &formatted, nil
	}
}

func stringPointerEquals(left *string, right *string) bool {
	if (left == nil && right != nil) || (left != nil && right == nil) {
		return false
	}

	if left == nil && right == nil {
		return true
	}

	return *left == *right
}

func intPointerEquals(left *int, right *int) bool {
	if (left == nil && right != nil) || (left != nil && right == nil) {
		return false
	}

	if left == nil && right == nil {
		return true
	}

	return *left == *right
}

func locationPointerEquals(left *Location, right *Location) bool {
	if (left == nil && right != nil) || (left != nil && right == nil) {
		return false
	}

	if left == nil && right == nil {
		return true
	}

	return left.File == right.File &&
		intPointerEquals(left.Line, right.Line) &&
		intPointerEquals(left.Column, right.Column)
}
