// Package fs is a thin wrapper around potential file-systems. By default, it is an abstraction over the `os` package
// from the standard library.
package fs

import (
	"os"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// Local is a local file-system. It wraps the default `os` package
type Local struct{}

// Open opens a file for further processing
func (l Local) Open(name string) (File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return f, nil
}
