package mocks

import (
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// Parser is a mocked implementation of 'cli.Parser'.
type Parser struct {
	MockParse            func(io.Reader) error
	MockIsTestCaseFailed func() bool
	MockNextTestCase     func() bool
	MockTestCaseID       func() string
}

// Parse either calls the configured mock of itself or returns an error if that doesn't exist.
func (p *Parser) Parse(reader io.Reader) error {
	if p.MockParse != nil {
		return p.MockParse(reader)
	}

	return errors.NewConfigurationError("MockParser was not configured")
}

// IsTestCaseFailed either calls the configured mock of itself or returns 'false' if that doesn't exist.
func (p *Parser) IsTestCaseFailed() bool {
	if p.MockIsTestCaseFailed != nil {
		return p.MockIsTestCaseFailed()
	}

	return false
}

// NextTestCase either calls the configured mock of itself or returns 'false' if that doesn't exist.
func (p *Parser) NextTestCase() bool {
	if p.MockNextTestCase != nil {
		return p.MockNextTestCase()
	}

	return false
}

// TestCaseID either calls the configured mock of itself or returns an empty string if that doesn't exist.
func (p *Parser) TestCaseID() string {
	if p.MockTestCaseID != nil {
		return p.MockTestCaseID()
	}

	return ""
}
