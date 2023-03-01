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

	"github.com/rwx-research/captain-cli"
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
	endpoint := "/api/test_suites/timing_manifest"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.NewInternalError("unable to construct HTTP request: %s", err)
	}

	queryValues := req.URL.Query()
	queryValues.Add("test_suite_identifier", testSuiteIdentifier)
	queryValues.Add("commit_sha", c.Provider.GetCommitSha())
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

func (c Client) registerTestResults(
	ctx context.Context,
	testSuite string,
	testResultsFiles []TestResultsFile,
) ([]TestResultsFile, error) {
	endpoint := "/api/test_suites/bulk_test_results"

	reqBody := struct {
		AttemptedBy         string            `json:"attempted_by"`
		Provider            string            `json:"provider"`
		BranchName          string            `json:"branch"`
		CommitMessage       *string           `json:"commit_message"`
		CommitSha           string            `json:"commit_sha"`
		TestSuiteIdentifier string            `json:"test_suite_identifier"`
		TestResultsFiles    []TestResultsFile `json:"test_results_files"`
		JobTags             map[string]any    `json:"job_tags"`
	}{
		AttemptedBy:         c.Provider.GetAttemptedBy(),
		Provider:            c.Provider.GetProviderName(),
		BranchName:          c.Provider.GetBranchName(),
		CommitSha:           c.Provider.GetCommitSha(),
		TestSuiteIdentifier: testSuite,
		TestResultsFiles:    testResultsFiles,
		JobTags:             c.Provider.GetJobTags(),
	}

	commitMessage := c.Provider.GetCommitMessage()
	if commitMessage != "" {
		reqBody.CommitMessage = &commitMessage
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
		TestResultsUploads []struct {
			ExternalID uuid.UUID `json:"external_identifier"`
			CaptainID  string    `json:"id"`
			UploadURL  string    `json:"upload_url"`
		} `json:"test_results_uploads"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, errors.NewInternalError(
			"unable to parse the response body. Endpoint was %q, Content-Type %q. Original Error: %s",
			endpoint,
			resp.Header.Get(headerContentType),
			err,
		)
	}

	for _, endpoint := range respBody.TestResultsUploads {
		parsedURL, err := url.Parse(endpoint.UploadURL)
		if err != nil {
			return nil, errors.NewInternalError("unable to parse S3 URL")
		}
		for i, testResultsFile := range testResultsFiles {
			if testResultsFile.ExternalID == endpoint.ExternalID {
				testResultsFiles[i].captainID = endpoint.CaptainID
				testResultsFiles[i].uploadURL = parsedURL
				break
			}
		}
	}

	return testResultsFiles, nil
}

func (c Client) updateTestResultsStatuses(
	ctx context.Context,
	testSuite string,
	testResultsFiles []TestResultsFile,
) error {
	endpoint := "/api/test_suites/bulk_test_results"

	type uploadStatus struct {
		CaptainID string `json:"id"`
		Status    string `json:"upload_status"`
	}

	reqBody := struct {
		TestSuiteIdentifier string         `json:"test_suite_identifier"`
		TestResultsFiles    []uploadStatus `json:"test_results_files"`
	}{}
	reqBody.TestSuiteIdentifier = testSuite

	for _, testResultsFile := range testResultsFiles {
		switch {
		case testResultsFile.s3uploadStatus < 300:
			reqBody.TestResultsFiles = append(reqBody.TestResultsFiles, uploadStatus{testResultsFile.captainID, "uploaded"})
		default:
			reqBody.TestResultsFiles = append(reqBody.TestResultsFiles, uploadStatus{testResultsFile.captainID, "upload_failed"})
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

// UploadTestResults uploads test results files to Captain.
// This method is not atomic - data-loss can occur silently. To verify that this operation was successful,
// the Captain database has to be queried manually.
func (c Client) UploadTestResults(
	ctx context.Context,
	testSuite string,
	testResultsFiles []TestResultsFile,
) ([]TestResultsUploadResult, error) {
	if testSuite == "" {
		return nil, errors.NewInputError("test suite name required")
	}

	fileSizeLookup := make(map[uuid.UUID]int64)
	for _, testResultsFile := range testResultsFiles {
		fileInfo, err := testResultsFile.FD.Stat()
		if err != nil {
			return nil, errors.NewSystemError("unable to determine file-size for %q", testResultsFile.FD.Name())
		}

		fileSizeLookup[testResultsFile.ExternalID] = fileInfo.Size()
	}

	testResultsFiles, err := c.registerTestResults(ctx, testSuite, testResultsFiles)
	if err != nil {
		return nil, err
	}

	uploadResults := make([]TestResultsUploadResult, 0)
	for i, testResultsFile := range testResultsFiles {
		if testResultsFile.uploadURL == nil {
			return nil, errors.NewInternalError("endpoint failed to return upload destination url")
		}
		req := http.Request{
			Method:        http.MethodPut,
			URL:           testResultsFile.uploadURL,
			Body:          testResultsFile.FD,
			ContentLength: fileSizeLookup[testResultsFile.ExternalID],
		}

		resp, err := c.RoundTrip(&req)
		if err != nil {
			c.Log.Warnf("unable to upload test results file to S3: %s", err)
			uploadResults = append(uploadResults, TestResultsUploadResult{
				OriginalPaths: testResultsFile.OriginalPaths,
				Uploaded:      false,
			})
			continue
		}
		defer resp.Body.Close()

		testResultsFiles[i].s3uploadStatus = resp.StatusCode
		uploadResults = append(uploadResults, TestResultsUploadResult{
			OriginalPaths: testResultsFile.OriginalPaths,
			Uploaded:      resp.StatusCode < 300,
		})
	}

	if err := c.updateTestResultsStatuses(ctx, testSuite, testResultsFiles); err != nil {
		c.Log.Warnf("unable to update test results file status: %s", err)
	}

	return uploadResults, nil
}
