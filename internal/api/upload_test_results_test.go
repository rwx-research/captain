package api_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Uploading Test Results", func() {
	var (
		apiClient          api.Client
		testResultsFiles   []api.TestResultsFile
		mockFile           *mocks.File
		mockRoundTripper   func(*http.Request) (*http.Response, error)
		mockRoundTripCalls int
	)

	BeforeEach(func() {
		mockRoundTripCalls = 0

		mockFile = new(mocks.File)
		mockFile.Reader = strings.NewReader("")

		testResultsFiles = []api.TestResultsFile{{
			ExternalID: uuid.New(),
			FD:         mockFile,
		}}
	})

	JustBeforeEach(func() {
		apiClientConfig := api.ClientConfig{Log: zap.NewNop().Sugar(), Provider: providers.GithubProvider{}}
		apiClient = api.Client{ClientConfig: apiClientConfig, RoundTrip: mockRoundTripper}
	})

	Context("under expected conditions", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the test results file
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(HaveSuffix("bulk_test_results"))
					Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", testResultsFiles[0].ExternalID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.String()).To(ContainSubstring(fmt.Sprintf("%d", GinkgoRandomSeed())))
					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.Path).To(HaveSuffix("bulk_test_results"))
					body, err := io.ReadAll(req.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(ContainSubstring(fmt.Sprintf(
						"%q,\"upload_status\":\"uploaded\"",
						"some-captain-identifier",
					)))
					resp.Body = io.NopCloser(strings.NewReader(""))
				default:
					Fail("too many HTTP calls")
				}

				mockRoundTripCalls++

				return &resp, nil
			}
		})

		It("registers, uploads, and updates the test result in sequence", func() {
			uploadResults, err := apiClient.UploadTestResults(context.Background(), "test suite id", testResultsFiles)
			Expect(uploadResults).To(HaveLen(1))
			Expect(uploadResults[0].Uploaded).To(Equal(true))
			Expect(err).To(Succeed())
		})
	})

	Context("with an error during test results file registration", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				return nil, errors.NewInternalError("Error")
			}
		})

		It("returns an error to the user", func() {
			uploadResults, err := apiClient.UploadTestResults(context.Background(), "test suite id", testResultsFiles)
			Expect(uploadResults).To(BeNil())
			Expect(err).ToNot(Succeed())
		})
	})

	Context("with an error from S3", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the test results file
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", testResultsFiles[0].ExternalID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					resp.Body = io.NopCloser(strings.NewReader(""))
					resp.StatusCode = 500
				case 2: // update status
					body, err := io.ReadAll(req.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(ContainSubstring(fmt.Sprintf(
						"%q,\"upload_status\":\"upload_failed\"",
						"some-captain-identifier",
					)))
					resp.Body = io.NopCloser(strings.NewReader(""))
				default:
					Fail("too many HTTP calls")
				}

				mockRoundTripCalls++

				return &resp, nil
			}
		})

		It("updates the upload status as failed", func() {
			uploadResults, err := apiClient.UploadTestResults(context.Background(), "test suite id", testResultsFiles)
			Expect(uploadResults).To(HaveLen(1))
			Expect(uploadResults[0].Uploaded).To(Equal(false))
			Expect(err).To(Succeed())
		})
	})

	Context("with an error while updating an test results file status", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var (
					resp http.Response
					err  error
				)

				switch mockRoundTripCalls {
				case 0: // registering the test results files
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", testResultsFiles[0].ExternalID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					err = errors.NewInternalError("Error")
				default:
					Fail("too many HTTP calls")
				}

				mockRoundTripCalls++

				return &resp, err
			}
		})

		It("does not return an error to the user", func() {
			uploadResults, err := apiClient.UploadTestResults(context.Background(), "test suite id", testResultsFiles)
			Expect(uploadResults).To(HaveLen(1))
			Expect(uploadResults[0].Uploaded).To(Equal(true))
			Expect(err).To(Succeed())
		})
	})
})
