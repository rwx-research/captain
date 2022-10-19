package mocks

import (
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// API is a mocked implementation of 'exec.Command'.
type Command struct {
	MockStart      func() error
	MockStdoutPipe func() (io.ReadCloser, error)
	MockWait       func() error
}

// Start either calls the configured mock of itself or returns an error if that doesn't exist.
func (c *Command) Start() error {
	if c.MockStart != nil {
		return c.MockStart()
	}

	return errors.NewConfigurationError("MockStart was not configured")
}

// StdoutPipe either calls the configured mock of itself or returns an error if that doesn't exist.
func (c *Command) StdoutPipe() (io.ReadCloser, error) {
	if c.MockStdoutPipe != nil {
		return c.MockStdoutPipe()
	}

	return nil, errors.NewConfigurationError("MockStdoutPipe was not configured")
}

// Wait either calls the configured mock of itself or returns an error if that doesn't exist.
func (c *Command) Wait() error {
	if c.MockWait != nil {
		return c.MockWait()
	}

	return errors.NewConfigurationError("MockWait was not configured")
}
