package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	// The rewritten path is absolute so the RWX agent can resolve it regardless of Captain's
	// working directory.
	wantPath := filepath.Join(tmp, preservedAttachmentsDir, "retry-1", "command-1", traceRel)
	if attachments[0].Path != wantPath {
		t.Fatalf("expected rewritten path %q, got %q", wantPath, attachments[0].Path)
	}
	if !filepath.IsAbs(attachments[0].Path) {
		t.Fatalf("expected an absolute rewritten path, got %q", attachments[0].Path)
	}

	contents, err := os.ReadFile(attachments[0].Path)
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

// TestPreserveAttachmentsEmitsAbsolutePathFromSubdirectory confirms the rewritten path is rooted at
// Captain's working directory even when that directory is a subdirectory (e.g. a monorepo package),
// so the RWX agent can resolve it without knowing where Captain ran.
func TestPreserveAttachmentsEmitsAbsolutePathFromSubdirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workingDir := filepath.Join(root, "packages", "app")

	traceRel := filepath.Join("test-results", "example-chromium", "trace.zip")
	if err := os.MkdirAll(filepath.Join(workingDir, filepath.Dir(traceRel)), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingDir, traceRel), []byte("trace-contents"), 0o600); err != nil {
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

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: workingDir}}
	if err := service.preserveAttachments(results, "original-attempt"); err != nil {
		t.Fatalf("preserveAttachments: %v", err)
	}

	attachments := decodeAttachments(t, results.Tests[0].Attempt.Meta["fileAttachments"])
	wantPath := filepath.Join(workingDir, preservedAttachmentsDir, "original-attempt", traceRel)
	if attachments[0].Path != wantPath {
		t.Fatalf("expected rewritten path %q rooted at the working directory, got %q", wantPath, attachments[0].Path)
	}

	if _, err := os.Stat(attachments[0].Path); err != nil {
		t.Fatalf("preserved attachment missing at rewritten path: %v", err)
	}
}

func decodeScreenshot(t *testing.T, raw any) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal screenshot: %v", err)
	}

	var screenshot map[string]any
	if err := json.Unmarshal(encoded, &screenshot); err != nil {
		t.Fatalf("unmarshal screenshot: %v", err)
	}

	return screenshot
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestPreserveAttachmentsRewritesScreenshotPaths confirms an attempt's failure screenshot and HTML
// snapshot are copied and rewritten. These live under meta["screenshot"] rather than
// meta["fileAttachments"], and they're the paths the RWX agent uploads screenshots from.
func TestPreserveAttachmentsRewritesScreenshotPaths(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// A framework writes a failure screenshot to a deterministic path derived from the test's name;
	// a re-invocation of the same test writes to that same path.
	imageRel := filepath.Join("cypress", "screenshots", "login.cy.js", "logs in (failed).png")
	htmlRel := filepath.Join("tmp", "snapshots", "login.html")
	writeFile(t, filepath.Join(tmp, imageRel), "image-contents")
	writeFile(t, filepath.Join(tmp, htmlRel), "html-contents")

	results := &v1.TestResults{
		Tests: []v1.Test{
			{
				Name: "logs in",
				Attempt: v1.TestAttempt{
					Meta: map[string]any{
						"screenshot": map[string]string{"image": imageRel, "html": htmlRel},
					},
				},
			},
		},
	}

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
	if err := service.preserveAttachments(results, filepath.Join("retry-1", "command-1")); err != nil {
		t.Fatalf("preserveAttachments: %v", err)
	}

	screenshot := decodeScreenshot(t, results.Tests[0].Attempt.Meta["screenshot"])

	for _, tc := range []struct{ key, rel, contents string }{
		{"image", imageRel, "image-contents"},
		{"html", htmlRel, "html-contents"},
	} {
		// The rewritten path is absolute so the RWX agent can resolve it regardless of Captain's
		// working directory.
		wantPath := filepath.Join(tmp, preservedAttachmentsDir, "retry-1", "command-1", tc.rel)
		if screenshot[tc.key] != wantPath {
			t.Fatalf("expected %s rewritten to %q, got %q", tc.key, wantPath, screenshot[tc.key])
		}

		contents, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("preserved %s missing: %v", tc.key, err)
		}
		if string(contents) != tc.contents {
			t.Fatalf("preserved %s contents mismatch: %q", tc.key, contents)
		}

		// Original must remain (copied, not moved) so framework-generated reports stay valid.
		if _, err := os.Stat(filepath.Join(tmp, tc.rel)); err != nil {
			t.Fatalf("original %s should remain after preservation: %v", tc.key, err)
		}
	}
}

// TestPreserveAttachmentsRewritesAbsoluteScreenshotPaths confirms a screenshot reported as an
// absolute path — the shape a reporter that resolves paths itself emits — is mirrored under the
// scope directory rather than being skipped or written outside the working directory.
func TestPreserveAttachmentsRewritesAbsoluteScreenshotPaths(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	imageRel := filepath.Join("cypress", "screenshots", "login.cy.js", "logs in (failed).png")
	imageAbs := filepath.Join(tmp, imageRel)
	writeFile(t, imageAbs, "image-contents")

	results := &v1.TestResults{
		Tests: []v1.Test{
			{
				Name: "logs in",
				Attempt: v1.TestAttempt{
					Meta: map[string]any{
						"screenshot": map[string]any{"image": imageAbs},
					},
				},
			},
		},
	}

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
	if err := service.preserveAttachments(results, "original-attempt"); err != nil {
		t.Fatalf("preserveAttachments: %v", err)
	}

	screenshot := decodeScreenshot(t, results.Tests[0].Attempt.Meta["screenshot"])
	wantPath := filepath.Join(tmp, preservedAttachmentsDir, "original-attempt", imageRel)
	if screenshot["image"] != wantPath {
		t.Fatalf("expected image rewritten to %q, got %q", wantPath, screenshot["image"])
	}

	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("preserved screenshot missing at rewritten path: %v", err)
	}
}

// TestPreserveAttachmentsRewritesPastAttemptScreenshots confirms preservation reaches past attempts.
// That's the case the mechanism exists for: the earlier attempt's screenshot is the one a later
// invocation overwrites.
func TestPreserveAttachmentsRewritesPastAttemptScreenshots(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	imageRel := filepath.Join("screenshots", "logs in (failed).png")
	writeFile(t, filepath.Join(tmp, imageRel), "image-contents")

	results := &v1.TestResults{
		Tests: []v1.Test{
			{
				Name:    "logs in",
				Attempt: v1.TestAttempt{Meta: map[string]any{}},
				PastAttempts: []v1.TestAttempt{
					{Meta: map[string]any{"screenshot": map[string]string{"image": imageRel}}},
				},
			},
		},
	}

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
	if err := service.preserveAttachments(results, "original-attempt"); err != nil {
		t.Fatalf("preserveAttachments: %v", err)
	}

	screenshot := decodeScreenshot(t, results.Tests[0].PastAttempts[0].Meta["screenshot"])
	wantPath := filepath.Join(tmp, preservedAttachmentsDir, "original-attempt", imageRel)
	if screenshot["image"] != wantPath {
		t.Fatalf("expected past attempt image rewritten to %q, got %q", wantPath, screenshot["image"])
	}
}

// TestPreserveAttachmentsPreservesBothMetaKeys confirms an attempt carrying both keys has both
// preserved, and that a key we don't recognize under meta["screenshot"] survives the round trip.
func TestPreserveAttachmentsPreservesBothMetaKeys(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	traceRel := filepath.Join("test-results", "example-retry1", "trace.zip")
	imageRel := filepath.Join("screenshots", "example (failed).png")
	writeFile(t, filepath.Join(tmp, traceRel), "trace-contents")
	writeFile(t, filepath.Join(tmp, imageRel), "image-contents")

	results := &v1.TestResults{
		Tests: []v1.Test{
			{
				Name: "example",
				Attempt: v1.TestAttempt{
					Meta: map[string]any{
						"fileAttachments": []fileAttachment{{Name: "trace", Path: traceRel}},
						"screenshot":      map[string]any{"image": imageRel, "takenAt": "2026-07-31T00:00:00Z"},
					},
				},
			},
		},
	}

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
	if err := service.preserveAttachments(results, "original-attempt"); err != nil {
		t.Fatalf("preserveAttachments: %v", err)
	}

	attachments := decodeAttachments(t, results.Tests[0].Attempt.Meta["fileAttachments"])
	wantTracePath := filepath.Join(tmp, preservedAttachmentsDir, "original-attempt", traceRel)
	if attachments[0].Path != wantTracePath {
		t.Fatalf("expected trace rewritten to %q, got %q", wantTracePath, attachments[0].Path)
	}

	screenshot := decodeScreenshot(t, results.Tests[0].Attempt.Meta["screenshot"])
	wantImagePath := filepath.Join(tmp, preservedAttachmentsDir, "original-attempt", imageRel)
	if screenshot["image"] != wantImagePath {
		t.Fatalf("expected image rewritten to %q, got %q", wantImagePath, screenshot["image"])
	}
	if screenshot["takenAt"] != "2026-07-31T00:00:00Z" {
		t.Fatalf("expected unrecognized screenshot keys to survive, got %v", screenshot["takenAt"])
	}
}

// TestPreserveAttachmentsSkipsMissingScreenshot confirms a missing screenshot file is left untouched
// rather than erroring, matching how a missing file attachment is handled.
func TestPreserveAttachmentsSkipsMissingScreenshot(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	missingRel := filepath.Join("screenshots", "gone (failed).png")
	results := &v1.TestResults{
		Tests: []v1.Test{
			{
				Name: "example",
				Attempt: v1.TestAttempt{
					Meta: map[string]any{
						"screenshot": map[string]string{"image": missingRel},
					},
				},
			},
		},
	}

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
	if err := service.preserveAttachments(results, "original-attempt"); err != nil {
		t.Fatalf("preserveAttachments should not error on missing files: %v", err)
	}

	screenshot := decodeScreenshot(t, results.Tests[0].Attempt.Meta["screenshot"])
	if screenshot["image"] != missingRel {
		t.Fatalf("expected missing screenshot path to be left unchanged, got %q", screenshot["image"])
	}
}

// TestPreserveAttachmentsToleratesScreenshotShapes confirms the shapes a reporter could plausibly
// emit are all left untouched rather than erroring. meta["screenshot"] is free-form JSON, so
// anything that isn't a string path under a key we recognize has to pass through unharmed.
func TestPreserveAttachmentsToleratesScreenshotShapes(t *testing.T) {
	t.Parallel()

	for name, screenshot := range map[string]any{
		"null":              nil,
		"array":             []any{"screenshots/example.png"},
		"number image":      map[string]any{"image": 42},
		"nested image":      map[string]any{"image": map[string]any{"path": "screenshots/example.png"}},
		"empty image path":  map[string]any{"image": ""},
		"only unknown keys": map[string]any{"takenAt": "2026-07-31T00:00:00Z"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			results := &v1.TestResults{
				Tests: []v1.Test{
					{
						Name:    "example",
						Attempt: v1.TestAttempt{Meta: map[string]any{"screenshot": screenshot}},
					},
				},
			}

			service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
			if err := service.preserveAttachments(results, "original-attempt"); err != nil {
				t.Fatalf("preserveAttachments should not error on %s: %v", name, err)
			}

			// Nothing was preservable, so nothing should have been written.
			if _, err := os.Stat(filepath.Join(tmp, preservedAttachmentsDir)); !os.IsNotExist(err) {
				t.Fatalf("expected no preservation directory for %s", name)
			}
		})
	}
}

// TestPreserveAttachmentsIgnoresUnrecognizedScreenshotShape confirms a meta["screenshot"] that isn't
// a map is left alone rather than erroring or being replaced.
func TestPreserveAttachmentsIgnoresUnrecognizedScreenshotShape(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	results := &v1.TestResults{
		Tests: []v1.Test{
			{
				Name: "example",
				Attempt: v1.TestAttempt{
					Meta: map[string]any{"screenshot": "screenshots/example.png"},
				},
			},
		},
	}

	service := Service{FileSystem: fixedWorkingDirFS{workingDir: tmp}}
	if err := service.preserveAttachments(results, "original-attempt"); err != nil {
		t.Fatalf("preserveAttachments should not error on an unrecognized shape: %v", err)
	}

	if results.Tests[0].Attempt.Meta["screenshot"] != "screenshots/example.png" {
		t.Fatalf("expected unrecognized screenshot meta to be untouched, got %v",
			results.Tests[0].Attempt.Meta["screenshot"])
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
