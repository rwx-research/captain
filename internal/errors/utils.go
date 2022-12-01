package errors

import (
	"github.com/pkg/errors"
)

// As is a wrapper around the standard library `errors.As`
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Adds a stack trace to an error without doing anything further
func WithStack(err error) error {
	return errors.WithStack(err)
}

// Wrap is similar to 'WithStack', but adds a message to the error
func Wrap(err error, msg string) error {
	return errors.Wrap(err, msg)
}

// Wrapf is similar to 'Wrap', but formats the message
func Wrapf(err error, msg string, a ...any) error {
	return errors.Wrapf(err, msg, a...)
}

// Unwraps err one level
func Unwrap(err error) error {
	return errors.Unwrap(err)
}
