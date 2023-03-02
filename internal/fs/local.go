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

func (l Local) Create(filePath string) (File, error) {
	f, err := os.Create(filePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return f, nil
}

func (l Local) CreateTemp(dir string, pattern string) (File, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return f, nil
}

// Open opens a file for further processing
func (l Local) Open(name string) (File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return f, nil
}

func (l Local) Glob(pattern string) ([]string, error) {
	filepaths, err := filepathx.Glob(pattern)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return filepaths, nil
}

func (l Local) GlobMany(patterns []string) ([]string, error) {
	pathSet := make(map[string]struct{})
	for _, pattern := range patterns {
		expandedPaths, err := l.Glob(pattern)
		if err != nil {
			return nil, errors.WithStack(err)
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

func (l Local) Remove(name string) error {
	err := os.Remove(name)
	return errors.WithStack(err)
}

func (l Local) Rename(oldname string, newname string) error {
	err := os.Rename(oldname, newname)
	return errors.WithStack(err)
}

func (l Local) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return info, nil
}

func (l Local) TempDir() string {
	return os.TempDir()
}
