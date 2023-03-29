package errors

const errorTemplate = `{{.Type}}: {{.Title}}

{{.Description}}
{{.Resolution}}

If issues persist please reach out to support@rwx.com for assistance.
`

type detailedError interface {
	Description() string
	Error() string
	Resolution() string
	Type() string
}

type templateVariables struct {
	Title       string
	Type        string
	Description string
	Resolution  string
}

func (t templateVariables) Validate() error {
	if t.Title == "" {
		return NewInternalError("error message is missing a title")
	}

	if t.Type == "" {
		return NewInternalError("error message is missing a type")
	}

	if t.Description == "" {
		return NewInternalError("error message is missing a Description")
	}

	return nil
}
