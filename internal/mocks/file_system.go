package mocks

import (
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
)

// FileSystem is a mocked implementation of 'cli.FileSystem'.
type FileSystem struct {
	MockOpen     func(name string) (fs.File, error)
	MockGlob     func(pattern string) ([]string, error)
	MockGlobMany func(patterns []string) ([]string, error)
}

// Open either calls the configured mock of itself or returns an error if that doesn't exist.
func (f *FileSystem) Open(name string) (fs.File, error) {
	if f.MockOpen != nil {
		return f.MockOpen(name)
	}

	return nil, errors.NewConfigurationError("MockOpen was not configured")
}

func (f *FileSystem) Glob(pattern string) ([]string, error) {
	if f.MockGlob != nil {
		return f.MockGlob(pattern)
	}

	return nil, errors.NewConfigurationError("MockGlob was not configured")
}

func (f *FileSystem) GlobMany(patterns []string) ([]string, error) {
	if f.MockGlob != nil {
		return f.MockGlob(patterns[0])
	}

	return nil, errors.NewConfigurationError("MockGlob was not configured")
}
