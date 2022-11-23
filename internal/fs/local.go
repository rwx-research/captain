// Package fs is a thin wrapper around potential file-systems. By default, it is an abstraction over the `os` package
// from the standard library.
package fs

import (
	"os"
	"sort"

	"github.com/yargevad/filepathx"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// Local is a local file-system. It wraps the default `os` package
type Local struct{}

// Open opens a file for further processing
func (l Local) Open(name string) (File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return f, nil
}

func (l Local) Glob(pattern string) ([]string, error) {
	filepaths, err := filepathx.Glob(pattern)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return filepaths, nil
}

func (l Local) GlobMany(patterns []string) ([]string, error) {
	pathSet := make(map[string]struct{})
	for _, pattern := range patterns {
		expandedPaths, err := l.Glob(pattern)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		for _, filePath := range expandedPaths {
			pathSet[filePath] = struct{}{}
		}
	}

	expandedPaths := make([]string, 0, len(pathSet))
	for expandedPath := range pathSet {
		expandedPaths = append(expandedPaths, expandedPath)
	}

	sort.Slice(expandedPaths, func(i, j int) bool {
		return expandedPaths[i] < expandedPaths[j]
	})

	return expandedPaths, nil
}
