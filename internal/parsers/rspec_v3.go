package parsers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// RSpecV3 is a RSpec parser for v3 test results.
type RSpecV3 struct{}

type rSpecV3Result struct {
	Version  string `json:"version"`
	Examples []struct {
		FullDescription string  `json:"full_description"`
		ID              string  `json:"id"`
		Runtime         float64 `json:"run_time"`
		Status          string  `json:"status"`
		PendingMessage  string  `json:"pending_message"`
		FilePath        string  `json:"file_path"`
		Exception       struct {
			Message string `json:"message"`
		} `json:"exception"`
	} `json:"examples"`
}

// Parse attempts to parse the provided byte-stream as an RSpec v3 test suite.
func (r *RSpecV3) Parse(content io.Reader) (map[string]testing.TestResult, error) {
	var RSpecResult rSpecV3Result

	if err := json.NewDecoder(content).Decode(&RSpecResult); err != nil {
		return nil, errors.NewInputError("unable to parse document as JSON: %s", err)
	}

	if !strings.HasPrefix(RSpecResult.Version, "v") {
		RSpecResult.Version = fmt.Sprintf("v%s", RSpecResult.Version)
	}

	if semver.Major(RSpecResult.Version) != "v3" {
		return nil, errors.NewInputError("provided JSON document is not a RSpec v3 test results")
	}

	results := make(map[string]testing.TestResult)

	for _, example := range RSpecResult.Examples {
		var status testing.TestStatus
		var statusMessage string

		switch example.Status {
		case "failed":
			status = testing.TestStatusFailed
			statusMessage = example.Exception.Message
		case "passed":
			status = testing.TestStatusSuccessful
		case "pending":
			status = testing.TestStatusPending
			statusMessage = example.PendingMessage
		}

		id := example.ID
		if id == "" {
			id = example.FullDescription
		}

		file := example.ID
		if file == "" {
			file = example.FilePath
		} else {
			file = regexp.MustCompile(`\.rb(:.+|\[.+\])$`).ReplaceAllString(file, ".rb")
		}

		results[id] = testing.TestResult{
			Description:   example.FullDescription,
			Duration:      time.Duration(math.Round(example.Runtime * float64(time.Second))),
			Status:        status,
			StatusMessage: statusMessage,
			Meta:          map[string]any{"id": id, "file": file},
		}
	}

	return results, nil
}
