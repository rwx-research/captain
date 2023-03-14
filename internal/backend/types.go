package backend

import (
	"context"
	"net/url"

	"github.com/google/uuid"

	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// Client is the interface of our API layer.
type Client interface {
	GetRunConfiguration(ctx context.Context, testSuiteIdentifier string) (RunConfiguration, error)
	GetTestTimingManifest(context.Context, string) ([]testing.TestFileTiming, error)
	UploadTestResults(context.Context, string, []TestResultsFile) ([]TestResultsUploadResult, error)
}

// ParserType is an enum holding possible parser types
type ParserType string

// ParserTypeRWX is the parser type of RWX's test results schema
const ParserTypeRWX = ParserType("rwx")

type QuarantinedTest struct {
	Test
	QuarantinedAt string `json:"quarantined_at"`
}

type RunConfiguration struct {
	GeneratedAt      string            `json:"generated_at,omitempty"`
	QuarantinedTests []QuarantinedTest `json:"quarantined_tests,omitempty"`
	FlakyTests       []Test            `json:"flaky_tests,omitempty"`
}

type Test struct {
	CompositeIdentifier string   `json:"composite_identifier"`
	IdentityComponents  []string `json:"identity_components"`
	StrictIdentity      bool     `json:"strict_identity"`
}

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

type TestResultsUploadResult struct {
	OriginalPaths []string
	Uploaded      bool
}
