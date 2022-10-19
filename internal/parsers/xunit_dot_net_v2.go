package parsers

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// XUnitDotNetV2 is an XUnit parser for v2 artifacts.
type XUnitDotNetV2 struct {
	result                   xUnitDotNetV2Assemblies
	assemblyIndex, testIndex int
}

type xUnitDotNetV2Assemblies struct {
	XMLName    xml.Name                `xml:"assemblies"`
	Assemblies []xUnitDotNetV2Assembly `xml:"assembly"`
}

type xUnitDotNetV2Assembly struct {
	Name  string              `xml:"name,attr"`
	Tests []xUnitDotNetV2Test `xml:"collection>test"`
}

type xUnitDotNetV2Test struct {
	Name   string `xml:"name,attr"`
	Result string `xml:"result,attr"`
}

// Parse attempts to parse the provided byte-stream as an XUnit test suite.
func (x *XUnitDotNetV2) Parse(content io.Reader) error {
	if err := xml.NewDecoder(content).Decode(&x.result); err != nil {
		return errors.NewInputError("unable to parse document as XML: %s", err)
	}

	return nil
}

func (x *XUnitDotNetV2) test() xUnitDotNetV2Test {
	return x.result.Assemblies[x.assemblyIndex].Tests[x.testIndex-1]
}

// IsTestCaseFailed returns whether or not the current test case has failed.
//
// Caution: this method will panic unless `NextTestCase` was previously called.
func (x *XUnitDotNetV2) IsTestCaseFailed() bool {
	return x.test().Result != "Pass"
}

// NextTestCase prepares the next test case for reading. It returns 'true' if this was successful and 'false' if there
// is no further test case to process.
//
// Caution: This method needs to be called before any other further data is read from the parser.
func (x *XUnitDotNetV2) NextTestCase() bool {
	x.testIndex++

	if len(x.result.Assemblies) == 0 {
		return false
	}

	if x.testIndex > len(x.result.Assemblies[x.assemblyIndex].Tests) {
		x.testIndex = 0
		x.assemblyIndex++

		if x.assemblyIndex == len(x.result.Assemblies) {
			return false
		}

		return x.NextTestCase()
	}

	return true
}

// TestCaseID returns the ID of the current test case.
//
// Caution: this method will panic unless `NextTestCase` was previously called.
func (x *XUnitDotNetV2) TestCaseID() string {
	assemblyName := x.result.Assemblies[x.assemblyIndex].Name
	return fmt.Sprintf("%s > %s", assemblyName, x.test().Name)
}
