package fs

import (
	"bytes"
	"io/fs"
	"os"
	"time"
)

type VirtualReadOnlyFile struct {
	*bytes.Reader
	FileName string
}

// Close is needed in order to satisfy the io.Closer interface
func (vf VirtualReadOnlyFile) Close() error {
	return nil
}

// IsDir is needed in order to satisfy the os.FileInfo interface
func (vf VirtualReadOnlyFile) IsDir() bool {
	return false
}

// Mode is needed in order to satisfy the os.FileInfo interface
func (vf VirtualReadOnlyFile) Mode() fs.FileMode {
	return fs.ModeIrregular
}

// ModTime is needed in order to satisfy the os.FileInfo interface
func (vf VirtualReadOnlyFile) ModTime() time.Time {
	return time.Now()
}

// Name is needed in order to satisfy the File interface from above.
func (vf VirtualReadOnlyFile) Name() string {
	return vf.FileName
}

// Stat is needed in order to satisfy the File interface from above.
func (vf VirtualReadOnlyFile) Stat() (os.FileInfo, error) {
	return vf, nil
}

// Sys is needed in order to satisfy the os.FileInfo interface
func (vf VirtualReadOnlyFile) Sys() any {
	return nil
}
