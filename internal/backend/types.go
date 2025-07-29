package backend

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/testing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// Client is the interface of our API layer.
type Client interface {
	GetRunConfiguration(ctx context.Context, testSuiteIdentifier string) (RunConfiguration, error)
	GetQuarantinedTests(ctx context.Context, testSuiteIdentifier string) ([]Test, error)
	GetTestTimingManifest(context.Context, string) ([]testing.TestFileTiming, error)
	UpdateTestResults(context.Context, string, v1.TestResults) ([]TestResultsUploadResult, error)
}

type QuarantinedTest struct {
	Test
	QuarantinedAt string `json:"quarantined_at"`
}

type RunConfiguration struct {
	GeneratedAt        string            `json:"generated_at,omitempty"`
	QuarantinedTests   []QuarantinedTest `json:"quarantined_tests,omitempty"`
	FlakyTests         []Test            `json:"flaky_tests,omitempty"`
	OrganizationSlug   string            `json:"organization_slug,omitempty"`
	IsSuiteQuarantined bool              `json:"is_suite_quarantined,omitempty"`
}

type Test struct {
	CompositeIdentifier string   `json:"composite_identifier"`
	IdentityComponents  []string `json:"identity_components"`
	StrictIdentity      bool     `json:"strict_identity"`
}

type TestResultsUploadResult struct {
	OriginalPaths []string
	Uploaded      bool
}
