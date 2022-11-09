package api

import (
	"net/url"

	"github.com/google/uuid"

	"github.com/rwx-research/captain-cli/internal/fs"
)

// TestResultsFile is a build- or test-artifact as defined by the Captain API.
type TestResultsFile struct {
	ExternalID     uuid.UUID  `json:"external_identifier"`
	FD             fs.File    `json:"-"`
	OriginalPath   string     `json:"original_path"`
	Parser         ParserType `json:"format"`
	uploadURL      *url.URL
	captainID      string
	s3uploadStatus int
}

// ParserType is an enum holding possible parser types
type ParserType string

// ParserTypeCypressXML is the parser type of Cypes results encoded in XML
const ParserTypeCypessXML = ParserType("cypress_junit_xml")

// ParserTypeJestJSON is the parser type of Jest results encoded in JSON
const ParserTypeJestJSON = ParserType("jest_json")

// ParserTypeJUnitXML is the parser type of JUnit results encoded in XML
const ParserTypeJUnitXML = ParserType("junit_xml")

// ParserTypeRSpecJSON is the parser type of RSpec results encoded in JSON
const ParserTypeRSpecJSON = ParserType("rspec_json")

// ParserTypeXUnitXML is the parser type of XUnit results encoded in XML
const ParserTypeXUnitXML = ParserType("xunit_dot_net_xml")
