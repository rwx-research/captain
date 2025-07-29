package remote

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

const (
	fileSizeThresholdBytes = 25 * 1024 * 1024
	truncationMessage      = "<truncated due to test results size>"
)

func (c Client) registerTestResults(
	ctx context.Context,
	testSuite string,
	testResultsFile TestResultsFile,
) (TestResultsFile, error) {
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
		TestResultsFiles:    []TestResultsFile{testResultsFile},
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
		return TestResultsFile{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return TestResultsFile{}, errors.NewInternalError(
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
		return TestResultsFile{}, errors.NewInternalError(
			"unable to parse the response body. Endpoint was %q, Content-Type %q. Original Error: %s",
			endpoint,
			resp.Header.Get(headerContentType),
			err,
		)
	}

	for _, endpoint := range respBody.TestResultsUploads {
		parsedURL, err := url.Parse(endpoint.UploadURL)
		if err != nil {
			return TestResultsFile{}, errors.NewInternalError("unable to parse S3 URL")
		}

		if testResultsFile.ExternalID == endpoint.ExternalID {
			testResultsFile.CaptainID = endpoint.CaptainID
			testResultsFile.UploadURL = parsedURL
			break
		}
	}

	return testResultsFile, nil
}

func (c Client) updateTestResultsStatuses(
	ctx context.Context,
	testSuite string,
	testResultsFile TestResultsFile,
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

	switch {
	case testResultsFile.S3uploadStatus < 300:
		reqBody.TestResultsFiles = append(reqBody.TestResultsFiles, uploadStatus{testResultsFile.CaptainID, "uploaded"})
	default:
		reqBody.TestResultsFiles = append(reqBody.TestResultsFiles, uploadStatus{testResultsFile.CaptainID, "upload_failed"})
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
) ([]backend.TestResultsUploadResult, error) {
	if testSuite == "" {
		return nil, errors.NewInputError("test suite name required")
	}

	id, err := c.NewUUID()
	if err != nil {
		return nil, c.logError(errors.NewInternalError("Unable to generate new UUID: %s", err))
	}

	testResultsFile, err := c.makeTestResultsFile(testResults, id)
	if err != nil {
		return nil, c.logError(err)
	}

	fileInfo, err := testResultsFile.FD.Stat()
	if err != nil {
		return nil, errors.NewSystemError("unable to determine file-size for %q", testResultsFile.FD.Name())
	}

	type stripFunc func(v1.TestResults) v1.TestResults
	for _, strip := range []stripFunc{c.stripDerivedFrom, c.stripPreviousAttempts, c.stripCurrentAttempts} {
		if fileInfo.Size() <= fileSizeThresholdBytes {
			break
		}

		testResults = strip(testResults)

		testResultsFile, err = c.makeTestResultsFile(testResults, id)
		if err != nil {
			return nil, c.logError(err)
		}

		fileInfo, err = testResultsFile.FD.Stat()
		if err != nil {
			return nil, errors.NewSystemError("unable to determine file-size for %q", testResultsFile.FD.Name())
		}
	}

	testResultsFile, err = c.registerTestResults(ctx, testSuite, testResultsFile)
	if err != nil {
		return nil, err
	}

	uploadResults := make([]backend.TestResultsUploadResult, 0)
	if testResultsFile.UploadURL == nil {
		return nil, errors.NewInternalError("endpoint failed to return upload destination url")
	}
	req := http.Request{
		Method:        http.MethodPut,
		URL:           testResultsFile.UploadURL,
		Body:          testResultsFile.FD,
		ContentLength: fileInfo.Size(),
	}

	resp, err := c.RoundTrip(&req)
	if err != nil {
		c.Log.Warnf("unable to upload test results file to S3: %s", err)
		uploadResults = append(uploadResults, backend.TestResultsUploadResult{
			OriginalPaths: uniqueStrings(testResultsFile.OriginalPaths),
			Uploaded:      false,
		})
	} else {
		testResultsFile.S3uploadStatus = resp.StatusCode
		uploadResults = append(uploadResults, backend.TestResultsUploadResult{
			OriginalPaths: uniqueStrings(testResultsFile.OriginalPaths),
			Uploaded:      resp.StatusCode < 300,
		})
		_ = resp.Body.Close()
	}

	if err := c.updateTestResultsStatuses(ctx, testSuite, testResultsFile); err != nil {
		c.Log.Warnf("unable to update test results file status: %s", err)
	}

	return uploadResults, nil
}

func (c Client) makeTestResultsFile(testResults v1.TestResults, externalID uuid.UUID) (TestResultsFile, error) {
	buf, err := json.Marshal(testResults)
	if err != nil {
		return TestResultsFile{}, errors.NewInternalError("Unable to output test results as JSON: %s", err)
	}

	f := fs.VirtualReadOnlyFile{
		Reader:   bytes.NewReader(buf),
		FileName: "rwx-test-results",
	}

	originalPaths := make([]string, len(testResults.DerivedFrom))
	for i, originalTestResult := range testResults.DerivedFrom {
		originalPaths[i] = originalTestResult.OriginalFilePath
	}

	testResultsFile := TestResultsFile{
		ExternalID:    externalID,
		FD:            f,
		OriginalPaths: originalPaths,
		Parser:        ParserTypeRWX,
	}

	return testResultsFile, nil
}

func (c Client) stripDerivedFrom(testResults v1.TestResults) v1.TestResults {
	c.Log.Warnf("removing original test result data from uploaded Captain test results due to content size threshold")
	cleanedDerivedFrom := make([]v1.OriginalTestResults, len(testResults.DerivedFrom))

	for i, originalTestResults := range testResults.DerivedFrom {
		cleanedDerivedFrom[i] = v1.OriginalTestResults{
			OriginalFilePath: originalTestResults.OriginalFilePath,
			GroupNumber:      originalTestResults.GroupNumber,
			Contents:         base64.StdEncoding.EncodeToString([]byte(truncationMessage)),
		}
	}

	return v1.TestResults{
		Framework:   testResults.Framework,
		Summary:     testResults.Summary,
		Tests:       testResults.Tests,
		OtherErrors: testResults.OtherErrors,
		DerivedFrom: cleanedDerivedFrom,
	}
}

func (c Client) stripPreviousAttempts(testResults v1.TestResults) v1.TestResults {
	c.Log.Warnf("removing previous test attempts from uploaded Captain test results due to content size threshold")

	for i, test := range testResults.Tests {
		for j, attempt := range test.PastAttempts {
			testResults.Tests[i].PastAttempts[j].Status = stripStatus(attempt.Status)
		}
	}

	return testResults
}

func (c Client) stripCurrentAttempts(testResults v1.TestResults) v1.TestResults {
	c.Log.Warnf("removing current test attempts from uploaded Captain test results due to content size threshold")

	for i, test := range testResults.Tests {
		if test.Attempt.Status.Backtrace != nil {
			testResults.Tests[i].Attempt.Status = stripStatus(test.Attempt.Status)
		}
	}

	return testResults
}

func stripStatus(status v1.TestStatus) v1.TestStatus {
	if status.Backtrace != nil {
		status.Backtrace = []string{truncationMessage}
	}

	if status.OriginalStatus != nil {
		s := stripStatus(*status.OriginalStatus)
		status.OriginalStatus = &s
	}

	return status
}
