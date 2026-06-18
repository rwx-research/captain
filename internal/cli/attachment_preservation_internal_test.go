package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// fixedWorkingDirFS is a real filesystem whose Getwd is pinned to a fixed directory, so tests can
// exercise attachment preservation against a temp dir without mutating the process's working
// directory (which would make them unsafe to run in parallel).
type fixedWorkingDirFS struct {
	fs.Local
	workingDir string
}

func (f fixedWorkingDirFS) Getwd() (string, error) {
	return f.workingDir, nil
}

func decodeAttachments(t *testing.T, raw any) []fileAttachment {
	t.Helper()

	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal attachments: %v", err)
	}

	var attachments []fileAttachment
	if err := json.Unmarshal(encoded, &attachments); err != nil {
		t.Fatalf("unmarshal attachments: %v", err)
	}

	return attachments
}

// TestPreserveAttachmentsRewritesPaths confirms that an attempt's attachment is copied to a unique,
// invocation-scoped location and its path is rewritten, while the original is left in place.
func TestPreserveAttachmentsRewritesPaths(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// A framework writes a trace to a deterministic per-retry path; a re-invocation would reuse it.
	traceRel := filepath.Join("test-results", "example-retry1", "trace.zip")
	if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(traceRel)), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, traceRel), []byte("trace-contents"), 0o600); err != nil {
		t.Fatal(err)
	}

	results := &v1.TestResults{
		Tests: []v1.Test{
			{
				Name: "example",
				Attempt: v1.TestAttempt{
					Meta: map[string]any{
						"fileAttachments": []fileAttachment{{Name: "trace", Path: traceRel}},
					},
				},
			},
		},
	}

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
	if err := service.preserveAttachments(results, filepath.Join("retry-1", "command-1")); err != nil {
		t.Fatalf("preserveAttachments: %v", err)
	}

	attachments := decodeAttachments(t, results.Tests[0].Attempt.Meta["fileAttachments"])
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}

	wantPrefix := filepath.Join(preservedAttachmentsDir, "retry-1", "command-1")
	if !strings.HasPrefix(attachments[0].Path, wantPrefix) {
		t.Fatalf("expected rewritten path under %q, got %q", wantPrefix, attachments[0].Path)
	}
	if attachments[0].Path == traceRel {
		t.Fatal("expected attachment path to be rewritten, but it was unchanged")
	}

	contents, err := os.ReadFile(filepath.Join(tmp, attachments[0].Path))
	if err != nil {
		t.Fatalf("preserved attachment missing: %v", err)
	}
	if string(contents) != "trace-contents" {
		t.Fatalf("preserved contents mismatch: %q", contents)
	}

	// Original must remain (copied, not moved) so e.g. the Playwright HTML report link stays valid.
	if _, err := os.Stat(filepath.Join(tmp, traceRel)); err != nil {
		t.Fatalf("original attachment should remain after preservation: %v", err)
	}
}

// TestPreserveAttachmentsSkipsMissingFiles confirms a missing source (already overwritten/removed) is
// left untouched rather than erroring.
func TestPreserveAttachmentsSkipsMissingFiles(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	missingRel := filepath.Join("test-results", "gone-retry1", "trace.zip")
	results := &v1.TestResults{
		Tests: []v1.Test{
			{
				Name: "example",
				Attempt: v1.TestAttempt{
					Meta: map[string]any{
						"fileAttachments": []fileAttachment{{Name: "trace", Path: missingRel}},
					},
				},
			},
		},
	}

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
	if err := service.preserveAttachments(results, "original-attempt"); err != nil {
		t.Fatalf("preserveAttachments should not error on missing files: %v", err)
	}

	attachments := decodeAttachments(t, results.Tests[0].Attempt.Meta["fileAttachments"])
	if attachments[0].Path != missingRel {
		t.Fatalf("expected missing attachment path to be left unchanged, got %q", attachments[0].Path)
	}
}
