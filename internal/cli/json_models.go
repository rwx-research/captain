package cli

import (
	"fmt"
	"time"

	"github.com/rwx-research/captain-cli/internal/testing"
)

type testResult struct {
	Description   string        `json:"description"`
	Duration      time.Duration `json:"duration"`
	Status        testStatus    `json:"status"`
	StatusMessage string        `json:"status_message,omitempty"`
}

type testStatus testing.TestStatus

func (s testStatus) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", testing.TestStatus(s).String())), nil
}
