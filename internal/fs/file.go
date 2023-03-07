package fs

import (
	"io"
	"os"
)

// File is a generic interface that represents a file that was opened on a file-system. It is modelled after the default
// 'os.File' from the standard library.
type File interface {
	io.ReadSeekCloser
	io.Writer
	Name() string
	Stat() (os.FileInfo, error)
	Sync() error
}

// TODO: replace with io/fs.File
type ReadOnlyFile interface {
	io.ReadSeekCloser
	Stat() (os.FileInfo, error)
	Name() string
}
