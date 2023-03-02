package mocks

import (
	"os"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
)

// FileSystem is a mocked implementation of 'cli.FileSystem'.
type FileSystem struct {
	MockCreate     func(filePath string) (fs.File, error)
	MockCreateTemp func(dir string, pattern string) (fs.File, error)
	MockOpen       func(name string) (fs.File, error)
	MockGlob       func(pattern string) ([]string, error)
	MockGlobMany   func(patterns []string) ([]string, error)
	MockRemove     func(name string) error
	MockRename     func(oldname string, newname string) error
	MockStat       func(name string) (os.FileInfo, error)
	MockTempDir    func() string
}

// Create either calls the configured mock of itself or returns an error if that doesn't exist.
func (f *FileSystem) Create(filePath string) (fs.File, error) {
	if f.MockCreate != nil {
		return f.MockCreate(filePath)
	}

	return nil, errors.NewConfigurationError("MockCreate was not configured")
}

// CreateTemp either calls the configured mock of itself or returns an error if that doesn't exist.
func (f *FileSystem) CreateTemp(dir string, pattern string) (fs.File, error) {
	if f.MockCreateTemp != nil {
		return f.MockCreateTemp(dir, pattern)
	}

	return nil, errors.NewConfigurationError("MockCreateTemp was not configured")
}

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

func (f *FileSystem) Remove(name string) error {
	if f.MockRemove != nil {
		return f.MockRemove(name)
	}

	return errors.NewConfigurationError("MockRemove was not configured")
}

func (f *FileSystem) Rename(oldname string, newname string) error {
	if f.MockRename != nil {
		return f.MockRename(oldname, newname)
	}

	return errors.NewConfigurationError("MockRename was not configured")
}

func (f *FileSystem) Stat(name string) (os.FileInfo, error) {
	if f.MockStat != nil {
		return f.MockStat(name)
	}

	return nil, errors.NewConfigurationError("MockStat was not configured")
}

func (f *FileSystem) TempDir() string {
	if f.MockTempDir != nil {
		return f.MockTempDir()
	}

	return "tmp"
}
