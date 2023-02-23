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
	Stat() (os.FileInfo, error)
	Name() string
}
