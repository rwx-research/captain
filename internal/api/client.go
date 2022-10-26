// Package api holds the main API client for Captain & supporting types. Overall, this should be a fairly transparent
// package, mapping HTTP calls to Go methods - however some endpoints are a bit more complex & require multiple calls
// back-and forth.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/rwx-research/captain-cli/internal/errors"
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
		// However, it turns out that the API expects us to upload artifacts to S3 directly as well.
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

// GetQuarantinedTestIDs returns a list of test identifiers that are marked as quarantined on Captain.
func (c Client) GetQuarantinedTestIDs(ctx context.Context) ([]string, error) {
	endpoint := "/api/organization/integrations/github/quarantined_tests"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	queryValues := req.URL.Query()
	queryValues.Add("account_name", c.AccountName)
	queryValues.Add("repository_name", c.RepositoryName)
	queryValues.Add("workflow_run_id", c.RunID)
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
		QuarantinedTests []struct {
			Identifier string `json:"identifier"`
		} `json:"quarantined_tests"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, errors.NewInternalError(
			"unable to parse the response body. Endpoint was %q, Content-Type %q. Original Error: %s",
			endpoint,
			resp.Header.Get(headerContentType),
			err,
		)
	}

	quarantinedTests := make([]string, len(respBody.QuarantinedTests))
	for i, quarantinedTest := range respBody.QuarantinedTests {
		quarantinedTests[i] = quarantinedTest.Identifier
	}

	return quarantinedTests, nil
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

func (c Client) registerArtifacts(ctx context.Context, artifacts []Artifact) (map[uuid.UUID]*url.URL, error) {
	endpoint := "/api/organization/integrations/github/bulk_artifacts"

	reqBody := struct {
		AccountName    string     `json:"account_name"`
		Artifacts      []Artifact `json:"artifacts"`
		JobName        string     `json:"job_name"`
		JobMatrix      *string    `json:"job_matrix"`
		RepositoryName string     `json:"repository_name"`
		RunAttempt     string     `json:"run_attempt"`
		RunID          string     `json:"run_id"`
	}{
		AccountName:    c.AccountName,
		Artifacts:      artifacts,
		RepositoryName: c.RepositoryName,
		JobName:        c.JobName,
		JobMatrix:      c.JobMatrix,
		RunAttempt:     c.RunAttempt,
		RunID:          c.RunID,
	}

	resp, err := c.postJSON(ctx, endpoint, reqBody)
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
		BulkArtifacts []struct {
			ExternalID uuid.UUID `json:"external_id"`
			UploadURL  string    `json:"upload_url"`
		} `json:"bulk_artifacts"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, errors.NewInternalError(
			"unable to parse the response body. Endpoint was %q, Content-Type %q. Original Error: %s",
			endpoint,
			resp.Header.Get(headerContentType),
			err,
		)
	}

	S3Endpoints := make(map[uuid.UUID]*url.URL)

	for _, endpoint := range respBody.BulkArtifacts {
		parsedURL, err := url.Parse(endpoint.UploadURL)
		if err != nil {
			return nil, errors.NewInternalError("unable to parse S3 URL")
		}

		S3Endpoints[endpoint.ExternalID] = parsedURL
	}

	return S3Endpoints, nil
}

func (c Client) updateArtifactStatuses(ctx context.Context, statuses map[uuid.UUID]int) error {
	endpoint := "/api/organization/integrations/github/bulk_artifacts/status"

	type uploadStatus struct {
		ExternalID uuid.UUID `json:"external_id"`
		Status     string    `json:"status"`
	}

	reqBody := struct {
		Artifacts []uploadStatus `json:"artifacts"`
	}{}

	for id, status := range statuses {
		switch {
		case status < 300:
			reqBody.Artifacts = append(reqBody.Artifacts, uploadStatus{id, "uploaded"})
		default:
			reqBody.Artifacts = append(reqBody.Artifacts, uploadStatus{id, "upload_failed"})
		}
	}

	resp, err := c.putJSON(ctx, endpoint, reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return errors.NewInternalError(
			"API backend encountered an error. Endpoint was %q, Status Code %d",
			endpoint,
			resp.StatusCode,
		)
	}

	return nil
}

// UploadArtifacts uploads artifacts to Captain.
// This method is not atomic - data-loss can occur silently. To verify that this operation was successful,
// the Captain database has to be queried manually.
func (c Client) UploadArtifacts(ctx context.Context, artifacts []Artifact) error {
	fileSizeLookup := make(map[uuid.UUID]int64)
	for _, artifact := range artifacts {
		fileInfo, err := artifact.FD.Stat()
		if err != nil {
			return errors.NewSystemError("unable to determine file-size for %q", artifact.OriginalPath)
		}

		fileSizeLookup[artifact.ExternalID] = fileInfo.Size()
	}

	S3Endpoints, err := c.registerArtifacts(ctx, artifacts)
	if err != nil {
		return err
	}

	uploadStatuses := make(map[uuid.UUID]int)

	for _, artifact := range artifacts {
		req := http.Request{
			Method:        http.MethodPut,
			URL:           S3Endpoints[artifact.ExternalID],
			Body:          artifact.FD,
			ContentLength: fileSizeLookup[artifact.ExternalID],
		}

		resp, err := c.RoundTrip(&req)
		if err != nil {
			c.Log.Warnf("unable to upload artifact to S3: %s", err)
			continue
		}
		defer resp.Body.Close()

		uploadStatuses[artifact.ExternalID] = resp.StatusCode
	}

	if err := c.updateArtifactStatuses(ctx, uploadStatuses); err != nil {
		c.Log.Warnf("unable to update artifact status: %s", err)
	}

	return nil
}
