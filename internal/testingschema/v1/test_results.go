// testingschema/v1 holds the implementation of V1 of RWX's test results schema:
// https://raw.githubusercontent.com/rwx-research/test-results-schema/main/v1.json
package v1

import (
	"encoding/json"

	"github.com/rwx-research/captain-cli/internal/errors"
)

type TestResults struct {
	Framework   Framework             `json:"framework"`
	Summary     Summary               `json:"summary"`
	Tests       []Test                `json:"tests"`
	OtherErrors []OtherError          `json:"otherErrors,omitempty"`
	DerivedFrom []OriginalTestResults `json:"derivedFrom,omitempty"`
}

func (tr TestResults) MarshalJSON() ([]byte, error) {
	type Alias TestResults

	json, err := json.Marshal(&struct {
		Schema string `json:"$schema"`
		Alias
	}{
		Schema: "https://raw.githubusercontent.com/rwx-research/test-results-schema/main/v1.json",
		Alias:  (Alias)(tr),
	})

	return json, errors.Wrap(err)
}

func (tr *TestResults) UnmarshalJSON(b []byte) error {
	type Alias TestResults
	var a struct {
		Schema string `json:"$schema"`
		Alias
	}

	if err := json.Unmarshal(b, &a); err != nil {
		return errors.Wrap(err)
	}
	if a.Schema != "https://raw.githubusercontent.com/rwx-research/test-results-schema/main/v1.json" {
		return errors.NewInputError("The parsed JSON is not the v1 RWX test results schema")
	}

	*tr = TestResults{
		Framework:   a.Framework,
		Summary:     a.Summary,
		Tests:       a.Tests,
		OtherErrors: a.OtherErrors,
		DerivedFrom: a.DerivedFrom,
	}

	return nil
}
