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

// FileTimingMatch represents the client file path matching the server test file timing.
// We store the client file path alongside the timing so we can ensure to only run the file paths
// originally provided by the client.
type FileTimingMatch struct {
	FileTiming     TestFileTiming
	ClientFilepath string
}

func (m FileTimingMatch) String() string {
	return fmt.Sprintf("'%s' (%s)", m.ClientFilepath, m.FileTiming.Duration)
}

func (m FileTimingMatch) Duration() time.Duration {
	return m.FileTiming.Duration
}
