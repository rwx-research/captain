package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

func (c Client) registerTestResults(
	ctx context.Context,
	testSuite string,
	testResultsFiles []TestResultsFile,
) ([]TestResultsFile, error) {
	endpoint := hostEndpointCompat(c, "/api/test_suites/bulk_test_results")

	reqBody := struct {
		AttemptedBy         string            `json:"attempted_by"`
		Provider            string            `json:"provider"`
		BranchName          string            `json:"branch"`
		CommitMessage       *string           `json:"commit_message"`
		CommitSha           string            `json:"commit_sha"`
		TestSuiteIdentifier string            `json:"test_suite_identifier"`
		TestResultsFiles    []TestResultsFile `json:"test_results_files"`
		Title               *string           `json:"title"`
		JobTags             map[string]any    `json:"job_tags"`
	}{
		AttemptedBy:         c.Provider.AttemptedBy,
		Provider:            c.Provider.ProviderName,
		BranchName:          c.Provider.BranchName,
		CommitSha:           c.Provider.CommitSha,
		TestSuiteIdentifier: testSuite,
		TestResultsFiles:    testResultsFiles,
		JobTags:             c.Provider.JobTags,
	}

	commitMessage := c.Provider.CommitMessage
	if commitMessage != "" {
		reqBody.CommitMessage = &commitMessage
	}

	title := c.Provider.Title
	if title != "" {
		reqBody.Title = &title
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
				testResultsFiles[i].CaptainID = endpoint.CaptainID
				testResultsFiles[i].UploadURL = parsedURL
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
	endpoint := hostEndpointCompat(c, "/api/test_suites/bulk_test_results")

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
		case testResultsFile.S3uploadStatus < 300:
			reqBody.TestResultsFiles = append(reqBody.TestResultsFiles, uploadStatus{testResultsFile.CaptainID, "uploaded"})
		default:
			reqBody.TestResultsFiles = append(reqBody.TestResultsFiles, uploadStatus{testResultsFile.CaptainID, "upload_failed"})
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

// UpdateTestResults uploads test results files to Captain.
// This method is not atomic - data-loss can occur silently. To verify that this operation was successful,
// the Captain database has to be queried manually.
func (c Client) UpdateTestResults(
	ctx context.Context,
	testSuite string,
	testResults v1.TestResults,
	adjustDerivedFromDueToContentLength bool,
) ([]backend.TestResultsUploadResult, error) {
	if testSuite == "" {
		return nil, errors.NewInputError("test suite name required")
	}

	id, err := c.NewUUID()
	if err != nil {
		return nil, c.logError(errors.NewInternalError("Unable to generate new UUID: %s", err))
	}

	buf, err := json.Marshal(testResults)
	if err != nil {
		return nil, c.logError(errors.NewInternalError("Unable to output test results as JSON: %s", err))
	}

	f := fs.VirtualReadOnlyFile{
		Reader:   bytes.NewReader(buf),
		FileName: "rwx-test-results",
	}

	originalPaths := make([]string, len(testResults.DerivedFrom))
	for i, originalTestResult := range testResults.DerivedFrom {
		originalPaths[i] = originalTestResult.OriginalFilePath
	}

	testResultsFiles := []TestResultsFile{{
		ExternalID:    id,
		FD:            f,
		OriginalPaths: originalPaths,
		Parser:        ParserTypeRWX,
	}}

	fileSizeLookup := make(map[uuid.UUID]int64)
	for _, testResultsFile := range testResultsFiles {
		fileInfo, err := testResultsFile.FD.Stat()
		if err != nil {
			return nil, errors.NewSystemError("unable to determine file-size for %q", testResultsFile.FD.Name())
		}

		fileSizeLookup[testResultsFile.ExternalID] = fileInfo.Size()

		// strip out derivedFrom when over 5MB
		if adjustDerivedFromDueToContentLength && fileInfo.Size() > 5242880 {
			c.Log.Warnf("removing original test result data from uploaded Captain test results due to content size threshold")
			cleanedDerivedFrom := make([]v1.OriginalTestResults, len(testResults.DerivedFrom))

			for i, originalTestResults := range testResults.DerivedFrom {
				cleanedDerivedFrom[i] = v1.OriginalTestResults{
					OriginalFilePath: originalTestResults.OriginalFilePath,
					GroupNumber:      originalTestResults.GroupNumber,
					Contents:         "W29taXR0ZWRd", // base64 encoded "[omitted]"
				}
			}
			testResultsWithoutDerivedFrom := v1.TestResults{
				Framework:   testResults.Framework,
				Summary:     testResults.Summary,
				Tests:       testResults.Tests,
				OtherErrors: testResults.OtherErrors,
				DerivedFrom: cleanedDerivedFrom,
			}
			return c.UpdateTestResults(ctx, testSuite, testResultsWithoutDerivedFrom, false)
		}
	}

	testResultsFiles, err = c.registerTestResults(ctx, testSuite, testResultsFiles)
	if err != nil {
		return nil, err
	}

	uploadResults := make([]backend.TestResultsUploadResult, 0)
	for i, testResultsFile := range testResultsFiles {
		if testResultsFile.UploadURL == nil {
			return nil, errors.NewInternalError("endpoint failed to return upload destination url")
		}
		req := http.Request{
			Method:        http.MethodPut,
			URL:           testResultsFile.UploadURL,
			Body:          testResultsFile.FD,
			ContentLength: fileSizeLookup[testResultsFile.ExternalID],
		}

		resp, err := c.RoundTrip(&req)
		if err != nil {
			c.Log.Warnf("unable to upload test results file to S3: %s", err)
			uploadResults = append(uploadResults, backend.TestResultsUploadResult{
				OriginalPaths: uniqueStrings(testResultsFile.OriginalPaths),
				Uploaded:      false,
			})
			continue
		}
		defer resp.Body.Close()

		testResultsFiles[i].S3uploadStatus = resp.StatusCode
		uploadResults = append(uploadResults, backend.TestResultsUploadResult{
			OriginalPaths: uniqueStrings(testResultsFile.OriginalPaths),
			Uploaded:      resp.StatusCode < 300,
		})
	}

	if err := c.updateTestResultsStatuses(ctx, testSuite, testResultsFiles); err != nil {
		c.Log.Warnf("unable to update test results file status: %s", err)
	}

	return uploadResults, nil
}
