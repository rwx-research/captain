package parsers

import (
	"encoding/xml"
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// JUnit is a JUnit parser.
// Note: this also happens to work for Cypress artifacts.
type JUnit struct {
	result jUnitTestSuite
	index  int
}

type jUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuites"`
	TestCases []jUnitTestCase `xml:"testsuite>testcase"`
}

type jUnitTestCase struct {
	Name    string    `xml:"name,attr"`
	Error   *struct{} `xml:"error"`
	Failure *struct{} `xml:"failure"`
}

// Parse attempts to parse the provided byte-stream as a JUnit test suite.
func (j *JUnit) Parse(content io.Reader) error {
	if err := xml.NewDecoder(content).Decode(&j.result); err != nil {
		return errors.NewInputError("unable to parse document as XML: %s", err)
	}

	return nil
}

func (j *JUnit) testCase() jUnitTestCase {
	return j.result.TestCases[j.index-1]
}

// IsTestCaseFailed returns whether or not the current test case has failed.
//
// Caution: this method will panic unless `NextTestCase` was previously called.
func (j *JUnit) IsTestCaseFailed() bool {
	return j.testCase().Error != nil || j.testCase().Failure != nil
}

// NextTestCase prepares the next test case for reading. It returns 'true' if this was successful and 'false' if there
// is no further test case to process.
//
// Caution: This method needs to be called before any other further data is read from the parser.
func (j *JUnit) NextTestCase() bool {
	j.index++

	return j.index <= len(j.result.TestCases)
}

// TestCaseID returns the ID of the current test case.
//
// Caution: this method will panic unless `NextTestCase` was previously called.
func (j *JUnit) TestCaseID() string {
	return j.testCase().Name
}
