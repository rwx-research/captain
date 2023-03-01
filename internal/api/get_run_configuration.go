package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rwx-research/captain-cli/internal/errors"
)

type Test struct {
	CompositeIdentifier string   `json:"composite_identifier"`
	IdentityComponents  []string `json:"identity_components"`
	StrictIdentity      bool     `json:"strict_identity"`
}

type QuarantinedTest struct {
	Test
	QuarantinedAt string `json:"quarantined_at"`
}

type RunConfiguration struct {
	GeneratedAt      string            `json:"generated_at"`
	QuarantinedTests []QuarantinedTest `json:"quarantined_tests"`
	FlakyTests       []Test            `json:"flaky_tests"`
}

// GetRunConfiguration returns the runtime configuration for the run command (e.g. quarantined and flaky tests)
func (c Client) GetRunConfiguration(
	ctx context.Context,
	testSuiteIdentifier string,
) (RunConfiguration, error) {
	endpoint := "/api/test_suites/run_configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return RunConfiguration{}, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	queryValues := req.URL.Query()
	queryValues.Add("test_suite_identifier", testSuiteIdentifier)
	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.RoundTrip(req)
	if err != nil {
		return RunConfiguration{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return RunConfiguration{}, errors.NewInternalError(
			"API backend encountered an error. Endpoint was %q, Status Code %d",
			endpoint,
			resp.StatusCode,
		)
	}

	runConfiguration := RunConfiguration{}
	if err := json.NewDecoder(resp.Body).Decode(&runConfiguration); err != nil {
		return RunConfiguration{}, errors.NewInternalError(
			"unable to parse the response body. Endpoint was %q, Content-Type %q. Original Error: %s",
			endpoint,
			resp.Header.Get(headerContentType),
			err,
		)
	}

	return runConfiguration, nil
}
