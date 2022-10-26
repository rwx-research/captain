package api

import (
	"github.com/google/uuid"

	"github.com/rwx-research/captain-cli/internal/fs"
)

// Artifact is a build- or test-artifact as defined by the Captain API.
type Artifact struct {
	ExternalID   uuid.UUID    `json:"external_id"`
	FD           fs.File      `json:"-"`
	Kind         ArtifactKind `json:"kind"`
	MimeType     string       `json:"mime_type"`
	Name         string       `json:"name"`
	OriginalPath string       `json:"original_path"`
	Parser       ParserType   `json:"parser"`
}

// ArtifactKind is an enum holding possible artifact kinds
//
// Deprecated: This will be removed in the future
type ArtifactKind string

// ArtifactKindTestResult is the artifact kind of test results
const ArtifactKindTestResult = ArtifactKind("test_results")

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
const ParserTypeXUnitXML = ParserType("xunit_dot_not_xml")
