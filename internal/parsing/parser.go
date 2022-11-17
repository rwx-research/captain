package parsing

import (
	"fmt"
	"io"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type ParseResultSentiment int

const (
	PositiveParseResultSentiment ParseResultSentiment = 1
	NeutralParseResultSentiment  ParseResultSentiment = 0
	NegativeParseResultSentiment ParseResultSentiment = -1
)

func (s ParseResultSentiment) String() string {
	switch s {
	case PositiveParseResultSentiment:
		return "Positive"
	case NeutralParseResultSentiment:
		return "Neutral"
	case NegativeParseResultSentiment:
		return "Negative"
	default:
		return fmt.Sprintf("Unknown (%d)", s)
	}
}

type ParseResult struct {
	// A positive sentiment means it's the preferred parser for a particular test results file.
	// A parser should attempt to parse the file and return an error if it fails.
	// If a parser can parse a file, it should use hueristics on the contents that it can parse
	// to determine how likely it is to be (or not to be) a pytest/Cypress/etc result in the case of
	// shared formats like JUnit.
	Sentiment   ParseResultSentiment
	TestResults v1.TestResults
	Parser      Parser
}

type Parser interface {
	Parse(io.Reader) (*ParseResult, error)
}
