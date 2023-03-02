package fs

import (
	"os"
)

// FileSystem is an abstraction over file-systems. This is implemented by the default `os` package and can also be used
// for mocking.
type FileSystem interface {
	Create(filePath string) (File, error)
	CreateTemp(dir string, pattern string) (File, error)
	Open(name string) (File, error)
	Glob(pattern string) ([]string, error)
	GlobMany(patterns []string) ([]string, error)
	Remove(name string) error
	Rename(oldname string, newname string) error
	Stat(name string) (os.FileInfo, error)
	TempDir() string
}
