package mocks

import (
	"io/fs"
	"time"
)

// FileInfo is a mocked implementation of `fs.FileInfo`
type FileInfo struct {
	Dir        bool
	FileName   string
	FileMode   fs.FileMode
	FileSize   int64
	ModifiedAt time.Time

	MockModTime func() time.Time
	MockMode    func() fs.FileMode
}

func (f FileInfo) Mode() fs.FileMode {
	return f.FileMode
}

func (f FileInfo) IsDir() bool {
	return f.Dir
}

func (f FileInfo) ModTime() time.Time {
	return f.ModifiedAt
}

func (f FileInfo) Name() string {
	return f.FileName
}

func (f FileInfo) Size() int64 {
	return f.FileSize
}

// Sys always returns nil.
func (f FileInfo) Sys() any {
	return nil
}
