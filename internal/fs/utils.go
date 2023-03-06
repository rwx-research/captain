package fs

import (
	"path/filepath"
	"strings"
)

// IsLocal is a copy of the `unixIsLocal` function introduced in Go 1.20
// See https://github.com/golang/go/blob/go1.20.1/src/path/filepath/path.go#L190
func IsLocal(path string) bool {
	if filepath.IsAbs(path) || path == "" {
		return false
	}
	hasDots := false
	for p := path; p != ""; {
		var part string
		part, p, _ = strings.Cut(p, "/")
		if part == "." || part == ".." {
			hasDots = true
			break
		}
	}
	if hasDots {
		path = filepath.Clean(path)
	}
	if path == ".." || strings.HasPrefix(path, "../") {
		return false
	}
	return true
}
