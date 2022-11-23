package api

import (
	"net/url"

	"github.com/google/uuid"

	"github.com/rwx-research/captain-cli/internal/fs"
)

// TestResultsFile is a build- or test-artifact as defined by the Captain API.
type TestResultsFile struct {
	ExternalID     uuid.UUID  `json:"external_identifier"`
	FD             fs.File    `json:"-"`
	OriginalPaths  []string   `json:"-"`
	Parser         ParserType `json:"format"`
	uploadURL      *url.URL
	captainID      string
	s3uploadStatus int
}

type TestResultsUploadResult struct {
	OriginalPaths []string
	Uploaded      bool
}

// ParserType is an enum holding possible parser types
type ParserType string

// ParserTypeRWX is the parser type of RWX's test results schema
const ParserTypeRWX = ParserType("rwx")
