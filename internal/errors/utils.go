package errors

import (
	"errors"

	"golang.org/x/xerrors"
)

// As is a wrapper around the standard library `errors.As`
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Wrap wraps an error without doing anything further - this is useful for collecting stack traces.
func Wrap(err error) error {
	if err == nil {
		return nil
	}

	return xerrors.Errorf("%w", err)
}

// WithMessage is similar to 'Wrap', but adds a message to the error
func WithMessage(msg string, a ...any) error {
	return xerrors.Errorf(msg, a...)
}
