package mocks

import (
	"io/fs"
	"os"
	"strings"
	"time"
)

// File is a mocked implementation of `os.File`, based on a common `bytes.Reader`
type File struct {
	*strings.Builder
	*strings.Reader

	MockModTime func() time.Time
	MockMode    func() fs.FileMode
	MockName    func() string
}

// Close will always return nil.
func (f *File) Close() error {
	return nil
}

// Mode either calls the configured mock of itself or returns `fs.ModeIrregular`
func (f *File) Mode() fs.FileMode {
	if f.MockMode != nil {
		return f.MockMode()
	}

	return fs.ModeIrregular
}

// IsDir will always return false.
func (f *File) IsDir() bool {
	return false
}

// ModTime either calls the configured mock of itself or returns `time.Now`
func (f *File) ModTime() time.Time {
	if f.MockModTime != nil {
		return f.MockModTime()
	}

	return time.Now()
}

// Name either calls the configured mock of itself or returns an empty string
func (f *File) Name() string {
	if f.MockName != nil {
		return f.MockName()
	}

	return ""
}

// Stat is a no-op. This mocked file implementation covers the `os.FileInfo` interface already.
func (f *File) Stat() (os.FileInfo, error) {
	return f, nil
}

// Sync always returns nil.
func (f *File) Sync() error {
	return nil
}

// Sys always returns nil.
func (f *File) Sys() any {
	return nil
}
