package mocks

import (
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

// Parser is a mocked implementation of 'parsing.Parser'.
type Parser struct {
	MockParse func(io.Reader) (*v1.TestResults, error)
}

// Parse either calls the configured mock of itself or returns an error if that doesn't exist.
func (p *Parser) Parse(reader io.Reader) (*v1.TestResults, error) {
	if p.MockParse != nil {
		return p.MockParse(reader)
	}

	return nil, errors.NewConfigurationError("MockParser was not configured")
}
