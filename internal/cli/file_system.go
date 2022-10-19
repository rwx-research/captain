package cli

import "github.com/rwx-research/captain-cli/internal/fs"

// FileSystem is an abstraction over file-systems. This is implemented by the default `os` package and can also be used
// for mocking.
type FileSystem interface {
	Open(name string) (fs.File, error)
}
