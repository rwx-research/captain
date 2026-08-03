package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// preservedAttachmentsDir is the working-directory-relative root under which Captain copies test
// attachment files (e.g. Playwright traces, failure screenshots). Frameworks write attachments to
// deterministic per-retry paths; when Captain re-invokes the test command for a targeted retry, those
// files are overwritten in place. Without preserving them, only the final invocation's attachment
// survives on disk, and it ends up associated with whichever attempt the merged results happen to
// reference that path from.
const preservedAttachmentsDir = "captain-attempts"

// screenshotMetaKeys are the entries under an attempt's meta["screenshot"] that hold filesystem
// paths. The RWX agent uploads the files at these paths, so they need the same per-invocation
// preservation as meta["fileAttachments"]. Any other key is left untouched.
var screenshotMetaKeys = []string{"image", "html"}

// fileAttachment mirrors the shape Captain stores under an attempt's meta["fileAttachments"]. We
// round-trip through JSON rather than depend on a framework-specific type, so the reporter emits the
// same {name, path} shape it otherwise would.
type fileAttachment struct {
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

// shouldPreserveAttachments reports whether Captain is running in an RWX context, where the RWX agent
// resolves and uploads attachment paths from the emitted results. Gating on this keeps local
// `captain run` users from getting rewritten paths or extra files in their working tree.
func shouldPreserveAttachments() bool {
	return os.Getenv("RWX_TEST_RESULTS") != ""
}

// preserveAttachments copies every attempt's attachment files into a unique, invocation-scoped
// location and rewrites the in-memory attachment paths to point at the copies. It must be called
// after an invocation's results are parsed but before the next invocation overwrites the files.
func (s Service) preserveAttachments(results *v1.TestResults, scope string) error {
	if results == nil {
		return nil
	}

	workingDir, err := s.FileSystem.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	for ti := range results.Tests {
		test := &results.Tests[ti]

		if err := s.preserveAttemptAttachments(&test.Attempt, scope, workingDir); err != nil {
			return err
		}

		for ai := range test.PastAttempts {
			if err := s.preserveAttemptAttachments(&test.PastAttempts[ai], scope, workingDir); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s Service) preserveAttemptAttachments(attempt *v1.TestAttempt, scope string, workingDir string) error {
	if attempt.Meta == nil {
		return nil
	}

	if err := s.preserveFileAttachments(attempt, scope, workingDir); err != nil {
		return err
	}

	return s.preserveScreenshot(attempt, scope, workingDir)
}

func (s Service) preserveFileAttachments(attempt *v1.TestAttempt, scope string, workingDir string) error {
	raw, ok := attempt.Meta["fileAttachments"]
	if !ok {
		return nil
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return errors.WithStack(err)
	}

	var attachments []fileAttachment
	if err := json.Unmarshal(encoded, &attachments); err != nil {
		// Not a recognized attachment shape; leave it untouched.
		return nil //nolint:nilerr
	}

	changed := false
	for i := range attachments {
		destAbs, preserved, err := s.preserveFile(attachments[i].Path, scope, workingDir)
		if err != nil {
			return err
		}
		if !preserved {
			continue
		}

		attachments[i].Path = destAbs
		changed = true
	}

	if changed {
		attempt.Meta["fileAttachments"] = attachments
	}

	return nil
}

// preserveScreenshot preserves the failure screenshot and HTML snapshot an attempt reports under
// meta["screenshot"]. This is a separate key from meta["fileAttachments"], and it's the one the RWX
// agent uploads screenshots from, so both need preserving for a targeted retry to keep each attempt
// pointed at its own files.
func (s Service) preserveScreenshot(attempt *v1.TestAttempt, scope string, workingDir string) error {
	raw, ok := attempt.Meta["screenshot"]
	if !ok {
		return nil
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return errors.WithStack(err)
	}

	// Decode into a generic map rather than a fixed struct so that keys we don't recognize survive
	// the round trip untouched.
	var screenshot map[string]any
	if err := json.Unmarshal(encoded, &screenshot); err != nil {
		// Not a recognized screenshot shape; leave it untouched.
		return nil //nolint:nilerr
	}

	changed := false
	for _, key := range screenshotMetaKeys {
		srcPath, ok := screenshot[key].(string)
		if !ok {
			continue
		}

		destAbs, preserved, err := s.preserveFile(srcPath, scope, workingDir)
		if err != nil {
			return err
		}
		if !preserved {
			continue
		}

		screenshot[key] = destAbs
		changed = true
	}

	if changed {
		attempt.Meta["screenshot"] = screenshot
	}

	return nil
}

// preserveFile copies srcPath into the invocation-scoped preservation directory and returns the
// absolute path of the copy. It reports false when there is nothing to preserve: an empty path, or a
// source that a later invocation has already overwritten or removed. Callers leave those unchanged.
func (s Service) preserveFile(srcPath string, scope string, workingDir string) (string, bool, error) {
	if srcPath == "" {
		return "", false, nil
	}

	srcAbs := srcPath
	if !filepath.IsAbs(srcAbs) {
		srcAbs = filepath.Join(workingDir, srcPath)
	}

	if _, err := s.FileSystem.Stat(srcAbs); err != nil {
		return "", false, nil //nolint:nilerr
	}

	destRel := filepath.Join(preservedAttachmentsDir, scope, relativeAttachmentPath(srcPath, srcAbs, workingDir))
	destAbs := filepath.Join(workingDir, destRel)

	if err := s.FileSystem.MkdirAll(filepath.Dir(destAbs), 0o750); err != nil {
		return "", false, errors.WithStack(err)
	}
	if err := copyFile(s.FileSystem, srcAbs, destAbs); err != nil {
		return "", false, errors.WithStack(err)
	}

	// Emit an absolute path. The RWX agent resolves these paths, and Captain's working directory is
	// not necessarily the agent's workspace root (e.g. a monorepo subdirectory), so a relative path
	// can't be resolved unambiguously on that side.
	return destAbs, true, nil
}

// relativeAttachmentPath derives a working-directory-relative path to mirror the source attachment
// under, so the preserved copy resolves the same way the original did.
func relativeAttachmentPath(srcPath, srcAbs, workingDir string) string {
	if !filepath.IsAbs(srcPath) {
		return srcPath
	}

	if rel, err := filepath.Rel(workingDir, srcAbs); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}

	return strings.TrimPrefix(srcAbs, string(filepath.Separator))
}
