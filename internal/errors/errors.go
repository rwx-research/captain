// Package errors is our internal errors package. It should be used in place of the standard "errors" package,
// "golang.org/x/xerrors", or "fmt.Errorf".
// This package ensures that all errors have a correct category & collect stack-traces.
package errors

import (
	"errors"

	"golang.org/x/xerrors"
)

// ErrConfiguration represent a configuration error. When used, it should ideally also point towards the configuration
// value that caused this error to occur.
type ErrConfiguration error

// ConfigurationError returns a new error of type "ErrConfiguration"
func ConfigurationError(msg string, a ...any) ErrConfiguration {
	return xerrors.Errorf(msg, a...)
}

// IsConfigurationError checks whether the error is a configuration error
func IsConfigurationError(err error) bool {
	var e ErrConfiguration
	return errors.As(err, &e)
}

// ErrInternal is an internal error. This error type should only be used if an end-user cannot act upon it and would
// need to reach out to us for support.
type ErrInternal error

// InternalError returns a new error of type "ErrInternal"
func InternalError(msg string, a ...any) ErrInternal {
	return xerrors.Errorf(msg, a...)
}

// IsInternalError check whether the error is an internal error
func IsInternalError(err error) bool {
	var e ErrInternal
	return errors.As(err, &e)
}

// ErrSystem is returned when the CLI encountered a system error. This is most likely either an error during file read
// or a network error.
type ErrSystem error

// SystemError returns a new error of type "ErrSystem"
func SystemError(msg string, a ...any) ErrSystem {
	return xerrors.Errorf(msg, a...)
}

// IsSystemError checks whether the error is a system error
func IsSystemError(err error) bool {
	var e ErrSystem
	return errors.As(err, &e)
}
