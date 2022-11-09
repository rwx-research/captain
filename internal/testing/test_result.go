// Package testing holds any domain logic and domain models related to testing.
package testing

import "time"

// TestResult is a generic representation of a test result
type TestResult struct {
	Description   string
	Duration      time.Duration
	Status        TestStatus
	StatusMessage string
	Meta          map[string]any
}
