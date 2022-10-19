package mocks

import (
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// Parser is a mocked implementation of 'cli.Parser'.
type Parser struct {
	MockParse func(io.Reader) (map[string]testing.TestResult, error)
}

// Parse either calls the configured mock of itself or returns an error if that doesn't exist.
func (p *Parser) Parse(reader io.Reader) (map[string]testing.TestResult, error) {
	if p.MockParse != nil {
		return p.MockParse(reader)
	}

	return nil, errors.NewConfigurationError("MockParser was not configured")
}
