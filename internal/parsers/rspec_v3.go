package parsers

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// RSpecV3 is a RSpec parser for v3 artifacts.
type RSpecV3 struct {
	result rSpecV3Result
	index  int
}

type rSpecV3Result struct {
	Version  string           `json:"version"`
	Examples []rSpecV3Example `json:"examples"`
}

type rSpecV3Example struct {
	FullDescription string `json:"full_description"`
	ID              string `json:"id"`
	Status          string `json:"status"`
}

// Parse attempts to parse the provided byte-stream as an RSpec v3 test suite.
func (r *RSpecV3) Parse(content io.Reader) error {
	if err := json.NewDecoder(content).Decode(&r.result); err != nil {
		return errors.NewInputError("unable to parse document as JSON: %s", err)
	}

	if !strings.HasPrefix(r.result.Version, "v") {
		r.result.Version = fmt.Sprintf("v%s", r.result.Version)
	}

	if semver.Major(r.result.Version) != "v3" {
		return errors.NewInputError("provided JSON document is not a RSpec v3 artifact")
	}

	return nil
}

func (r *RSpecV3) example() rSpecV3Example {
	return r.result.Examples[r.index-1]
}

// IsTestCaseFailed returns whether or not the current test case has failed.
//
// Caution: this method will panic unless `NextTestCase` was previously called.
func (r *RSpecV3) IsTestCaseFailed() bool {
	return r.example().Status == "failed"
}

// NextTestCase prepares the next test case for reading. It returns 'true' if this was successful and 'false' if there
// is no further test case to process.
//
// Caution: This method needs to be called before any other further data is read from the parser.
func (r *RSpecV3) NextTestCase() bool {
	r.index++

	return r.index <= len(r.result.Examples)
}

// TestCaseID returns the ID of the current test case.
//
// Caution: this method will panic unless `NextTestCase` was previously called.
func (r *RSpecV3) TestCaseID() string {
	return r.example().ID
}
