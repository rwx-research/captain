package parsers

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// XUnitDotNetV2 is an XUnit parser for v2 test results.
type XUnitDotNetV2 struct{}

type xUnitDotNetV2Assemblies struct {
	XMLName    xml.Name `xml:"assemblies"`
	Assemblies []struct {
		Name  string `xml:"name,attr"`
		Tests []struct {
			Name           string  `xml:"name,attr"`
			Result         string  `xml:"result,attr"`
			FailureMessage string  `xml:"failure>message"`
			SkipReason     string  `xml:"reason"`
			Time           float64 `xml:"time,attr"`
		} `xml:"collection>test"`
	} `xml:"assembly"`
}

var assemblyNameRegexp = regexp.MustCompile(fmt.Sprintf(`[^%s]+$`, string(os.PathSeparator)))

// Parse attempts to parse the provided byte-stream as an XUnit test suite.
func (x *XUnitDotNetV2) Parse(content io.Reader) ([]testing.TestResult, error) {
	var assemblies xUnitDotNetV2Assemblies

	if err := xml.NewDecoder(content).Decode(&assemblies); err != nil {
		return nil, errors.NewInputError("unable to parse document as XML: %s", err)
	}

	results := make([]testing.TestResult, 0)
	for _, assembly := range assemblies.Assemblies {
		for _, assemblyTest := range assembly.Tests {
			var status testing.TestStatus
			var statusMessage string

			switch assemblyTest.Result {
			case "Pass":
				status = testing.TestStatusSuccessful
			case "Fail":
				status = testing.TestStatusFailed
				statusMessage = assemblyTest.FailureMessage
			case "Skip":
				status = testing.TestStatusPending
				statusMessage = assemblyTest.SkipReason
			}

			assemblyName := assemblyNameRegexp.FindString(assembly.Name)
			results = append(results, testing.TestResult{
				Description:   assemblyTest.Name,
				Duration:      time.Duration(math.Round(assemblyTest.Time * float64(time.Second))),
				Status:        status,
				StatusMessage: statusMessage,
				Meta:          map[string]any{"assembly": assemblyName},
			})
		}
	}

	return results, nil
}
