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
	MockGetwd      func() (string, error)
	MockGlob       func(pattern string) ([]string, error)
	MockGlobMany   func(patterns []string) ([]string, error)
	MockMkdirAll   func(string, os.FileMode) error
	MockMkdirTemp  func(string, string) (string, error)
	MockOpen       func(name string) (fs.File, error)
	MockOpenFile   func(name string, flag int, perm os.FileMode) (fs.File, error)
	MockRemove     func(name string) error
	MockRemoveAll  func(path string) error
	MockRename     func(oldname string, newname string) error
	MockStat       func(name string) (os.FileInfo, error)
	MockTempDir    func() string
}

// Create either calls the configured mock of itself or returns an error if that doesn't exist.
func (f *FileSystem) Create(filePath string) (fs.File, error) {
	if f.MockCreate != nil {
		return f.MockCreate(filePath)
	}

	return nil, errors.NewInternalError("MockCreate was not configured")
}

func (f *FileSystem) Getwd() (string, error) {
	if f.MockGetwd != nil {
		return f.MockGetwd()
	}

	return "", errors.NewInternalError("MockGetwd was not configured")
}

// CreateTemp either calls the configured mock of itself or returns an error if that doesn't exist.
func (f *FileSystem) CreateTemp(dir string, pattern string) (fs.File, error) {
	if f.MockCreateTemp != nil {
		return f.MockCreateTemp(dir, pattern)
	}

	return nil, errors.NewInternalError("MockCreateTemp was not configured")
}

func (f *FileSystem) Glob(pattern string) ([]string, error) {
	if f.MockGlob != nil {
		return f.MockGlob(pattern)
	}

	return nil, errors.NewInternalError("MockGlob was not configured")
}

func (f *FileSystem) GlobMany(patterns []string) ([]string, error) {
	if f.MockGlob != nil {
		return f.MockGlob(patterns[0])
	}

	return nil, errors.NewInternalError("MockGlob was not configured")
}

func (f *FileSystem) Open(name string) (fs.File, error) {
	if f.MockOpen != nil {
		return f.MockOpen(name)
	}

	return nil, errors.NewInternalError("MockOpen was not configured")
}

func (f *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (fs.File, error) {
	if f.MockOpenFile != nil {
		return f.MockOpenFile(name, flag, perm)
	}

	return nil, errors.NewInternalError("MockOpenFile was not configured")
}

func (f *FileSystem) MkdirAll(path string, perm os.FileMode) error {
	if f.MockMkdirAll != nil {
		return f.MockMkdirAll(path, perm)
	}

	return errors.NewInternalError("MockMkdirAll was not configured")
}

func (f *FileSystem) MkdirTemp(dir, pattern string) (string, error) {
	if f.MockMkdirTemp != nil {
		return f.MockMkdirTemp(dir, pattern)
	}

	return "", errors.NewInternalError("MockMkdirTemp was not configured")
}

func (f *FileSystem) Remove(name string) error {
	if f.MockRemove != nil {
		return f.MockRemove(name)
	}

	return errors.NewInternalError("MockRemove was not configured")
}

func (f *FileSystem) RemoveAll(path string) error {
	if f.MockRemoveAll != nil {
		return f.MockRemoveAll(path)
	}

	return errors.NewInternalError("MockRemoveAll was not configured")
}

func (f *FileSystem) Rename(oldname string, newname string) error {
	if f.MockRename != nil {
		return f.MockRename(oldname, newname)
	}

	return errors.NewInternalError("MockRename was not configured")
}

func (f *FileSystem) Stat(name string) (os.FileInfo, error) {
	if f.MockStat != nil {
		return f.MockStat(name)
	}

	return nil, errors.NewInternalError("MockStat was not configured")
}

func (f *FileSystem) TempDir() string {
	if f.MockTempDir != nil {
		return f.MockTempDir()
	}

	return "tmp"
}
