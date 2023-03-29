package mocks

import "github.com/rwx-research/captain-cli/internal/errors"

// API is a mocked implementation of 'exec.Command'.
type Command struct {
	MockStart func() error
	MockWait  func() error
}

// Start either calls the configured mock of itself or returns an error if that doesn't exist.
func (c *Command) Start() error {
	if c.MockStart != nil {
		return c.MockStart()
	}

	return errors.NewInternalError("MockStart was not configured")
}

// Wait either calls the configured mock of itself or returns an error if that doesn't exist.
func (c *Command) Wait() error {
	if c.MockWait != nil {
		return c.MockWait()
	}

	return errors.NewInternalError("MockWait was not configured")
}
