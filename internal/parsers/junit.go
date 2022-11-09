package parsers

import (
	"encoding/xml"
	"io"
	"math"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// JUnit is a JUnit parser.
// Note: this also happens to work for Cypress test results.
type JUnit struct{}

type jUnitTestSuite struct {
	XMLName   xml.Name `xml:"testsuites"`
	TestCases []struct {
		Name    string        `xml:"name,attr"`
		Error   *jUnitMessage `xml:"error"`
		Failure *jUnitMessage `xml:"failure"`
		Skipped *jUnitMessage `xml:"skipped"`
		Time    float64       `xml:"time,attr"`
	} `xml:"testsuite>testcase"`
}

type jUnitMessage struct {
	Message string `xml:"message,attr"`
}

// Parse attempts to parse the provided byte-stream as a JUnit test suite.
func (j *JUnit) Parse(content io.Reader) (map[string]testing.TestResult, error) {
	var testSuite jUnitTestSuite

	if err := xml.NewDecoder(content).Decode(&testSuite); err != nil {
		return nil, errors.NewInputError("unable to parse document as XML: %s", err)
	}

	results := make(map[string]testing.TestResult)
	for _, testCase := range testSuite.TestCases {
		status := testing.TestStatusSuccessful
		statusMessages := make([]string, 0)

		if testCase.Failure != nil {
			status = testing.TestStatusFailed
			statusMessages = append(statusMessages, testCase.Failure.Message)
		}

		if testCase.Error != nil {
			status = testing.TestStatusFailed
			statusMessages = append(statusMessages, testCase.Error.Message)
		}

		if testCase.Skipped != nil {
			status = testing.TestStatusPending
			statusMessages = []string{testCase.Skipped.Message}
		}

		results[testCase.Name] = testing.TestResult{
			Description:   testCase.Name,
			Duration:      time.Duration(math.Round(testCase.Time * float64(time.Second))),
			Status:        status,
			StatusMessage: strings.Join(statusMessages, "\n"),
		}
	}

	return results, nil
}
