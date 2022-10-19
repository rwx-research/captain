package errors

import (
	"errors"

	"golang.org/x/xerrors"
)

// As is a wrapper around the standard library `errors.As`
func As(err error, target any) bool {
	return errors.As(err, target)
}

// IsRWXError reports whether the error originates from this package.
func IsRWXError(err error) bool {
	if _, ok := AsConfigurationError(err); ok {
		return true
	}

	if _, ok := AsExecutionError(err); ok {
		return true
	}

	if _, ok := AsInputError(err); ok {
		return true
	}

	if _, ok := AsInternalError(err); ok {
		return true
	}

	if _, ok := AsSystemError(err); ok {
		return true
	}

	return false
}

// Wrap wraps an error without doing anything further - this is useful for collecting stack traces.
func Wrap(err error) error {
	return xerrors.Errorf("%w", err)
}

// WithMessage is similar to 'Wrap', but adds a message to the error
func WithMessage(msg string, a ...any) error {
	return xerrors.Errorf(msg, a...)
}
