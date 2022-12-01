package errors

import (
	"github.com/pkg/errors"
)

// As is a wrapper around the standard library `errors.As`
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Wrap an error without doing anything further - this is useful for collecting stack traces.
func WithStack(err error) error {
	return errors.WithStack(err)
}

// WithMessage is similar to 'Wrap', but adds a message to the error
func WithMessage(err error, msg string) error {
	return errors.WithMessage(err, msg)
}

// WithMessagef is similar to 'WithMessage', but formats the message
func WithMessagef(err error, msg string, a ...any) error {
	return errors.WithMessagef(err, msg, a...)
}
