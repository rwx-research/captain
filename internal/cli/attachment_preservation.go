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
// attachment files (e.g. Playwright traces). Frameworks write attachments to deterministic per-retry
// paths; when Captain re-invokes the test command for a targeted retry, those files are overwritten
// in place. Without preserving them, only the final invocation's attachment survives on disk, and it
// ends up associated with whichever attempt the merged results happen to reference that path from.
const preservedAttachmentsDir = "captain-attempts"

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
		srcPath := attachments[i].Path
		if srcPath == "" {
			continue
		}

		srcAbs := srcPath
		if !filepath.IsAbs(srcAbs) {
			srcAbs = filepath.Join(workingDir, srcPath)
		}

		// The file may already have been overwritten or removed by a later invocation; skip if missing.
		if _, err := s.FileSystem.Stat(srcAbs); err != nil {
			continue
		}

		destRel := filepath.Join(preservedAttachmentsDir, scope, relativeAttachmentPath(srcPath, srcAbs, workingDir))
		destAbs := filepath.Join(workingDir, destRel)

		if err := s.FileSystem.MkdirAll(filepath.Dir(destAbs), 0o750); err != nil {
			return errors.WithStack(err)
		}
		if err := copyFile(s.FileSystem, srcAbs, destAbs); err != nil {
			return errors.WithStack(err)
		}

		// Emit an absolute path. The RWX agent resolves these paths, and Captain's working
		// directory is not necessarily the agent's workspace root (e.g. a monorepo subdirectory),
		// so a relative path can't be resolved unambiguously on that side.
		attachments[i].Path = destAbs
		changed = true
	}

	if changed {
		attempt.Meta["fileAttachments"] = attachments
	}

	return nil
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
