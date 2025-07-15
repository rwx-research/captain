// Package remote holds the main API client for Captain & supporting types. Overall, this should be a fairly transparent
// package, mapping HTTP calls to Go methods - however some endpoints are a bit more complex & require multiple calls
// back-and forth.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// Client is the main client for the Captain API.
type Client struct {
	ClientConfig
	RoundTrip func(*http.Request) (*http.Response, error)
}

// NewClient is the preferred constructor for the API client. It makes sure that the configuration is valid & necessary
// defaults are applied.
func NewClient(cfg ClientConfig) (Client, error) {
	cfg = cfg.WithDefaults()

	if err := cfg.Validate(); err != nil {
		return Client{}, err
	}

	client := &http.Client{}

	roundTrip := func(req *http.Request) (*http.Response, error) {
		// This is a bit hacky. In theory, this roundtripper should solely be used for accessing Captain's own API.
		// However, it turns out that the API expects us to upload test results to S3 directly as well.
		// We re-use the same roundtripper here primarily to make mocking easier. Alternatives would be to
		// a) have two different roundtrippers (or one for each configured host)
		// b) have the API client instantiate another API client
		// c) move all of this sequental logic out of the API layer
		// None of these options are great - having this special case for the S3 upload seems the least bad (given
		// that there is only a single occurrence)
		if !strings.HasSuffix(req.URL.Host, "amazonaws.com") {
			req.URL.Scheme = "https"
			if cfg.Insecure {
				req.URL.Scheme = "http"
			}

			req.URL.Host = cfg.Host
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.Token))
			req.Header.Set("User-Agent", fmt.Sprintf("captain-cli/%s", strings.TrimPrefix(captain.Version, "v")))
		}

		if cfg.Debug {
			hasBody := req.Body != nil
			dump, _ := httputil.DumpRequest(req, hasBody)
			sanitizedDump := bearerTokenRegexp.ReplaceAll(dump, []byte("<redacted>"))
			cfg.Log.Debugf("Executing following HTTP request:\n\n%s\n", sanitizedDump)
		}

		resp, err := client.Do(req)
		if err != nil {
			return resp, errors.NewSystemError("unable to perform HTTP request to %q: %s", req.URL, err)
		}

		if cfg.Debug {
			dump, _ := httputil.DumpResponse(resp, true)
			sanitizedDump := setCookieHeaderRegexp.ReplaceAll(dump, []byte("Set-Cookie: <redacted>"))
			cfg.Log.Debugf("Received following response:\n\n%s\n", sanitizedDump)
		}

		return resp, nil
	}

	return Client{cfg, roundTrip}, nil
}

func (c Client) GetTestTimingManifest(
	ctx context.Context,
	testSuiteIdentifier string,
) ([]testing.TestFileTiming, error) {
	endpoint := hostEndpointCompat(c, "/api/test_suites/timing_manifest")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	queryValues := req.URL.Query()
	queryValues.Add("test_suite_identifier", testSuiteIdentifier)
	queryValues.Add("commit_sha", c.Provider.CommitSha)
	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.NewInternalError(
			"API backend encountered an error. Endpoint was %q, Status Code %d",
			endpoint,
			resp.StatusCode,
		)
	}

	respBody := struct {
		FileTimings []testing.TestFileTiming `json:"file_timings"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, errors.NewInternalError(
			"unable to parse the response body. Endpoint was %q, Content-Type %q. Original Error: %s",
			endpoint,
			resp.Header.Get(headerContentType),
			err,
		)
	}

	return respBody.FileTimings, nil
}

func (c Client) logError(err error) error {
	c.Log.Errorf(err.Error())
	return err
}

func (c Client) postJSON(ctx context.Context, endpoint string, body any) (*http.Response, error) {
	encodedBody, err := json.Marshal(body)
	if err != nil {
		return nil, errors.NewInternalError("unable to construct JSON object for request: %s", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(encodedBody))
	if err != nil {
		return nil, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	req.Header.Set(headerContentType, contentTypeJSON)

	resp, err := c.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c Client) putJSON(ctx context.Context, endpoint string, body any) (*http.Response, error) {
	encodedBody, err := json.Marshal(body)
	if err != nil {
		return nil, errors.NewInternalError("unable to construct JSON object for request: %s", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewBuffer(encodedBody))
	if err != nil {
		return nil, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	req.Header.Set(headerContentType, contentTypeJSON)

	resp, err := c.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetRunConfiguration returns the runtime configuration for the run command (e.g. quarantined and flaky tests)
func (c Client) GetRunConfiguration(
	ctx context.Context,
	testSuiteIdentifier string,
) (backend.RunConfiguration, error) {
	endpoint := hostEndpointCompat(c, "/api/test_suites/run_configuration")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return backend.RunConfiguration{}, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	queryValues := req.URL.Query()
	queryValues.Add("test_suite_identifier", testSuiteIdentifier)
	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.RoundTrip(req)
	if err != nil {
		return backend.RunConfiguration{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return backend.RunConfiguration{}, errors.NewInternalError(
			"API backend encountered an error. Endpoint was %q, Status Code %d",
			endpoint,
			resp.StatusCode,
		)
	}

	runConfiguration := backend.RunConfiguration{}
	if err := json.NewDecoder(resp.Body).Decode(&runConfiguration); err != nil {
		return backend.RunConfiguration{}, errors.NewInternalError(
			"unable to parse the response body. Endpoint was %q, Content-Type %q. Original Error: %s",
			endpoint,
			resp.Header.Get(headerContentType),
			err,
		)
	}

	return runConfiguration, nil
}

// GetQuarantinedTests returns only the list of quarantined tests
func (c Client) GetQuarantinedTests(
	ctx context.Context,
	testSuiteIdentifier string,
) ([]backend.Test, error) {
	endpoint := hostEndpointCompat(c, "/api/test_suites/quarantined_tests")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	queryValues := req.URL.Query()
	queryValues.Add("test_suite_identifier", testSuiteIdentifier)
	req.URL.RawQuery = queryValues.Encode()

	resp, err := c.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.NewInternalError(
			"API backend encountered an error. Endpoint was %q, Status Code %d",
			endpoint,
			resp.StatusCode,
		)
	}

	respBody := struct {
		QuarantinedTests []backend.Test `json:"quarantined_tests"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, errors.NewInternalError(
			"unable to parse the response body. Endpoint was %q, Content-Type %q. Original Error: %s",
			endpoint,
			resp.Header.Get(headerContentType),
			err,
		)
	}

	return respBody.QuarantinedTests, nil
}

func (c Client) GetIdentityRecipes(ctx context.Context) ([]byte, error) {
	endpoint := hostEndpointCompat(c, "/api/recipes")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	resp, err := c.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.NewInternalError(
			"API backend encountered an error. Endpoint was %q, Status Code %d",
			endpoint,
			resp.StatusCode,
		)
	}

	buffer := bytes.NewBuffer(make([]byte, 0))
	_, err = buffer.ReadFrom(resp.Body)
	if err != nil {
		return nil, errors.NewInternalError(
			"Unable to read HTTP response body from API. Endpoint was %q, Status Code %d",
			endpoint,
			resp.StatusCode,
		)
	}

	return buffer.Bytes(), nil
}

// TODO(TS): Remove this once we're no longer testing against versions that use captain.build
func hostEndpointCompat(c Client, endpoint string) string {
	remoteHost := c.ClientConfig.Host

	if !strings.Contains(remoteHost, "cloud") {
		return endpoint
	}
	return "/captain" + endpoint
}
