// Package fs is a thin wrapper around potential file-systems. By default, it is an abstraction over the `os` package
// from the standard library.
package fs

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/yargevad/filepathx"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// Local is a local file-system. It wraps the default `os` package
type Local struct{}

func (l Local) Create(path string) (File, error) {
	if err := l.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, errors.WithStack(err)
	}

	f, err := os.Create(path)
	return f, errors.WithStack(err)
}

func (l Local) Getwd() (string, error) {
	dir, err := os.Getwd()
	return dir, errors.WithStack(err)
}

func (l Local) CreateTemp(dir string, pattern string) (File, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return f, nil
}

func (l Local) Glob(pattern string) ([]string, error) {
	filepaths, err := filepathx.Glob(pattern)
	return filepaths, errors.WithStack(err)
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

func (l Local) MkdirAll(path string, perm os.FileMode) error {
	return errors.WithStack(os.MkdirAll(path, perm))
}

func (l Local) MkdirTemp(dir, pattern string) (string, error) {
	dir, err := os.MkdirTemp(dir, pattern)
	return dir, errors.WithStack(err)
}

// Open opens a file for reading. Use OpenFile to open a file with different permissions
func (l Local) Open(name string) (File, error) {
	f, err := os.Open(name)
	return f, errors.WithStack(err)
}

func (l Local) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	f, err := os.OpenFile(name, flag, perm)
	return f, errors.WithStack(err)
}

func (l Local) Remove(name string) error {
	return errors.WithStack(os.Remove(name))
}

func (l Local) RemoveAll(path string) error {
	return errors.WithStack(os.RemoveAll(path))
}

func (l Local) Rename(oldname string, newname string) error {
	return errors.WithStack(os.Rename(oldname, newname))
}

func (l Local) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)
	return info, errors.WithStack(err)
}

func (l Local) TempDir() string {
	return os.TempDir()
}
