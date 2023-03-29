package errors

import (
	"strings"
	"text/template"

	"github.com/mitchellh/go-wordwrap"
	"github.com/pkg/errors"
)

// As is a wrapper around the standard library `errors.As`
func As(err error, target any) bool {
	return errors.As(err, target)
}

// decorate returns a "pretty-printed" error message for end-users.
func decorate(err detailedError) string {
	t, parserErr := template.New("error").Parse(errorTemplate)
	if parserErr != nil {
		// TODO: Don't discard parserErr here
		return err.Error()
	}

	vars := templateVariables{
		Title:       err.Error(),
		Description: wordwrap.WrapString(err.Description(), 80),
		Resolution:  wordwrap.WrapString(err.Resolution(), 80),
		Type:        err.Type(),
	}

	if validationErr := vars.Validate(); validationErr != nil {
		// TODO: Don't discard validationErr here
		return err.Error()
	}

	var buf strings.Builder
	if renderErr := t.Execute(&buf, vars); renderErr != nil {
		// TODO: Don't discard renderErr here
		return err.Error()
	}

	return buf.String()
}

// Is is a wrapper around the standard library `errors.Is`
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// WithDecoration returns a generic (i.e. unwrapped / no stack-trace) error, but decorated
func WithDecoration(e error) error {
	var err detailedError

	if ok := As(e, &err); !ok {
		return e
	}

	return errors.New(decorate(err))
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
