package cli

import "io"

// Parser is the interface a potential parser needs to implement.
type Parser interface {
	Parse(io.Reader) error

	IsTestCaseFailed() bool
	NextTestCase() bool
	TestCaseID() string
}
