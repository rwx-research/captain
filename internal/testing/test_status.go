package testing

// TestStatus is the status of a test case.
type TestStatus int

const (
	TestStatusUnknown TestStatus = iota
	TestStatusSuccessful
	TestStatusFailed
	TestStatusPending
)

// String returns the string representation of a status.
func (s TestStatus) String() string {
	switch s {
	case TestStatusSuccessful:
		return "successful"
	case TestStatusFailed:
		return "failed"
	case TestStatusPending:
		return "pending"
	case TestStatusUnknown:
		fallthrough
	default:
		return "unknown"
	}
}
