package api

// QuarantinedTestCase is a test that has been identified within Captain as quarantined
type QuarantinedTestCase struct {
	CompositeIdentifier string   `json:"composite_identifier"`
	IdentityComponents  []string `json:"identity_components"`
	StrictIdentity      bool     `json:"strict_identity"`
}
