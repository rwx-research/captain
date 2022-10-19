// Package errors is our internal errors package. It should be used in place of the standard "errors" package,
// "golang.org/x/xerrors", or "fmt.Errorf".
// This package ensures that all errors have a correct category & collect stack-traces.
package errors

import "golang.org/x/xerrors"

// ConfigurationError represent a configuration error. When used, it should ideally also point towards the configuration
// value that caused this error to occur.
type ConfigurationError error

// NewConfigurationError returns a new ConfigurationError
func NewConfigurationError(msg string, a ...any) ConfigurationError {
	return xerrors.Errorf(msg, a...)
}

// AsConfigurationError checks whether the error is a configuration error
func AsConfigurationError(err error) (ConfigurationError, bool) {
	var e ConfigurationError
	ok := As(err, &e)
	return e, ok
}

// ExecutionError is an error that was encountered during the execution of a different task. Specifically, this is being
// used with the `captain run` command, which executes a build- or test-suite as a sub-process.
// Execution errors can store an optional error-code.
type ExecutionError struct {
	E    error
	Code int
}

// NewExecutionError returns a new ExecutionError
func NewExecutionError(code int, msg string, a ...any) ExecutionError {
	return ExecutionError{Code: code, E: xerrors.Errorf(msg, a...)}
}

// AsExecutionError checks whether the error is an execution error.
func AsExecutionError(err error) (ExecutionError, bool) {
	var e ExecutionError
	ok := As(err, &e)
	return e, ok
}

// Error returns the error message of this error
func (e ExecutionError) Error() string {
	return e.E.Error()
}

// InputError is an error caused by user input
type InputError error

// InputError returns a new InputError
func NewInputError(msg string, a ...any) InputError {
	return xerrors.Errorf(msg, a...)
}

// AsInputError checks whether the error is an input error
func AsInputError(err error) (InputError, bool) {
	var e InputError
	ok := As(err, &e)
	return e, ok
}

// InternalError is an internal error. This error type should only be used if an end-user cannot act upon it and would
// need to reach out to us for support.
type InternalError error

// InternalError returns a new InternalError
func NewInternalError(msg string, a ...any) InternalError {
	return xerrors.Errorf(msg, a...)
}

// AsInternalError checks whether the error is an internal error
func AsInternalError(err error) (InternalError, bool) {
	var e InternalError
	ok := As(err, &e)
	return e, ok
}

// SystemError is returned when the CLI encountered a system error. This is most likely either an error during file read
// or a network error.
type SystemError error

// SystemError returns a new SystemError
func NewSystemError(msg string, a ...any) SystemError {
	return xerrors.Errorf(msg, a...)
}

// AsSystemError checks whether the error is a system error
func AsSystemError(err error) (SystemError, bool) {
	var e SystemError
	ok := As(err, &e)
	return e, ok
}
