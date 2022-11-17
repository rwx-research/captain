package testing

import (
	"fmt"
	"time"
)

// TestFileTiming is an estimated runtime duration for a test file based off of historical runs recorded by Captain
type TestFileTiming struct {
	Filepath string        `json:"file_path"`
	Duration time.Duration `json:"duration_in_nanoseconds"`
}

func (t TestFileTiming) String() string {
	return fmt.Sprintf("'%s' (%s)", t.Filepath, t.Duration)
}
