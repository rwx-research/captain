package remote

import (
	"net/url"

	"github.com/google/uuid"

	"github.com/rwx-research/captain-cli/internal/fs"
)

// ParserType is an enum holding possible parser types
type ParserType string

// ParserTypeRWX is the parser type of RWX's test results schema
const ParserTypeRWX = ParserType("rwx")

// TestResultsFile is a build- or test-artifact as defined by the Captain API.
type TestResultsFile struct {
	ExternalID     uuid.UUID       `json:"external_identifier"`
	FD             fs.ReadOnlyFile `json:"-"`
	OriginalPaths  []string        `json:"-"`
	Parser         ParserType      `json:"format"`
	UploadURL      *url.URL
	CaptainID      string
	S3uploadStatus int
}
