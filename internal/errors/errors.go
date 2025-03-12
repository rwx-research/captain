// Package errors is our internal errors package. It should be used in place of the standard "errors" package.
// This package ensures that all errors have a correct category & collect stack-traces.
package errors

import "github.com/pkg/errors"

// ConfigurationError represent a configuration error. When used, it should ideally also point towards the configuration
// value that caused this error to occur.
type ConfigurationError struct {
	desc string
	err  error
	fix  string
}

func (e ConfigurationError) Error() string {
	return e.err.Error()
}

func (e ConfigurationError) Description() string {
	return e.desc
}

func (e ConfigurationError) Resolution() string {
	return e.fix
}

func (e ConfigurationError) Type() string {
	return "Invalid configuration"
}

// NewConfigurationError returns a new ConfigurationError
func NewConfigurationError(title, desc, fix string) error {
	return WithStack(ConfigurationError{err: errors.New(title), desc: desc, fix: fix})
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

func (e ExecutionError) Error() string {
	return e.E.Error()
}

// NewExecutionError returns a new ExecutionError
func NewExecutionError(code int, msg string, a ...any) error {
	return WithStack(ExecutionError{Code: code, E: errors.Errorf(msg, a...)})
}

// AsExecutionError checks whether the error is an execution error.
func AsExecutionError(err error) (ExecutionError, bool) {
	var e ExecutionError
	ok := As(err, &e)
	return e, ok
}

// InputError is an error caused by user input
type InputError struct {
	E error
}

func (e InputError) Error() string {
	return e.E.Error()
}

// InputError returns a new InputError
func NewInputError(msg string, a ...any) error {
	return WithStack(InputError{errors.Errorf(msg, a...)})
}

// AsInputError checks whether the error is an input error
func AsInputError(err error) (InputError, bool) {
	var e InputError
	ok := As(err, &e)
	return e, ok
}

// InternalError is an internal error. This error type should only be used if an end-user cannot act upon it and would
// need to reach out to us for support.
type InternalError struct {
	E error
}

func (e InternalError) Error() string {
	return e.E.Error()
}

// InternalError returns a new InternalError
func NewInternalError(msg string, a ...any) error {
	return WithStack(InternalError{errors.Errorf(msg, a...)})
}

// AsInternalError checks whether the error is an internal error
func AsInternalError(err error) (InternalError, bool) {
	var e InternalError
	ok := As(err, &e)
	return e, ok
}

// SystemError is returned when the CLI encountered a system error. This is most likely either an error during file read
// or a network error.
type SystemError struct {
	E error
}

func (e SystemError) Error() string {
	return e.E.Error()
}

// SystemError returns a new SystemError
func NewSystemError(msg string, a ...any) error {
	return WithStack(SystemError{errors.Errorf(msg, a...)})
}

// AsSystemError checks whether the error is a system error
func AsSystemError(err error) (SystemError, bool) {
	var e SystemError
	ok := As(err, &e)
	return e, ok
}

// RetryError is an error related to retries. This error type is used in conjunction with the --fail-on-misconfigured-retry flag.
type RetryError struct {
	E error
}

func (e RetryError) Error() string {
	return e.E.Error()
}

// RetryError returns a new RetryError
func NewRetryError(msg string, a ...any) error {
	return WithStack(RetryError{errors.Errorf(msg, a...)})
}

// AsRetryError checks whether the error is a retry error
func AsRetryError(err error) (RetryError, bool) {
	var e RetryError
	ok := As(err, &e)
	return e, ok
}

// CacheError is returned when the CLI encountered a cache error. It's intended that the application catches these errors and
// handles them gracefully
type CacheError struct {
	E error
}

func (e CacheError) Error() string {
	return e.E.Error()
}

// CacheError returns a new CacheError
func NewCacheError(msg string, a ...any) error {
	return WithStack(CacheError{errors.Errorf(msg, a...)})
}

// AsCacheError checks whether the error is a cache error
func AsCacheError(err error) (CacheError, bool) {
	var e CacheError
	ok := As(err, &e)
	return e, ok
}

// DuplicateTestIDError is an error caused by duplicate test IDs
type DuplicateTestIDError struct {
	E error
}

func (e DuplicateTestIDError) Error() string {
	return e.E.Error()
}

// DuplicateTestIDError returns a new DuplicateTestIDError
func NewDuplicateTestIDError(msg string, a ...any) error {
	return WithStack(DuplicateTestIDError{errors.Errorf(msg, a...)})
}

// AsDuplicateTestIDError checks whether the error is a duplicate test ID error
func AsDuplicateTestIDError(err error) (DuplicateTestIDError, bool) {
	var e DuplicateTestIDError
	ok := As(err, &e)
	return e, ok
}
