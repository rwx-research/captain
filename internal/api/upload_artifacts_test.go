package api_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Uploading Artifacts", func() {
	var (
		apiClient          api.Client
		artifacts          []api.Artifact
		mockRoundTripper   func(*http.Request) (*http.Response, error)
		mockRoundTripCalls int
	)

	BeforeEach(func() {
		mockRoundTripCalls = 0

		fd0, err := os.Create(filepath.Join(GinkgoT().TempDir(), "a0.json"))
		Expect(err).ToNot(HaveOccurred())

		artifacts = []api.Artifact{{
			ExternalID: uuid.New(),
			FD:         fd0,
		}}
	})

	JustBeforeEach(func() {
		apiClient = api.Client{ClientConfig: api.ClientConfig{Log: zap.NewNop().Sugar()}, RoundTrip: mockRoundTripper}
	})

	Context("under expected conditions", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the artifact
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(HaveSuffix("bulk_artifacts"))
					Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"bulk_artifacts\":[{\"external_id\":%q,\"upload_url\":\"%d\"}]}",
						artifacts[0].ExternalID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.String()).To(ContainSubstring(fmt.Sprintf("%d", GinkgoRandomSeed())))
					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.Path).To(HaveSuffix("status"))
					body, err := io.ReadAll(req.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(ContainSubstring(fmt.Sprintf(
						"%q,\"status\":\"uploaded\"",
						artifacts[0].ExternalID,
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
			Expect(apiClient.UploadArtifacts(context.Background(), artifacts)).To(Succeed())
		})
	})

	Context("with an error during artifact registration", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				return nil, errors.InternalError("Error")
			}
		})

		It("returns an error to the user", func() {
			Expect(apiClient.UploadArtifacts(context.Background(), artifacts)).ToNot(Succeed())
		})
	})

	Context("with an error from S3", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the artifact
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"bulk_artifacts\":[{\"external_id\":%q,\"upload_url\":\"%d\"}]}",
						artifacts[0].ExternalID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					resp.Body = io.NopCloser(strings.NewReader(""))
					resp.StatusCode = 500
				case 2: // update status
					body, err := io.ReadAll(req.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(ContainSubstring(fmt.Sprintf(
						"%q,\"status\":\"upload_failed\"",
						artifacts[0].ExternalID,
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
			Expect(apiClient.UploadArtifacts(context.Background(), artifacts)).To(Succeed())
		})
	})

	Context("with an error while updating an artifact status", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var (
					resp http.Response
					err  error
				)

				switch mockRoundTripCalls {
				case 0: // registering the artifact
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"bulk_artifacts\":[{\"external_id\":%q,\"upload_url\":\"%d\"}]}",
						artifacts[0].ExternalID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					err = errors.InternalError("Error")
				default:
					Fail("too many HTTP calls")
				}

				mockRoundTripCalls++

				return &resp, err
			}
		})

		It("does not return an error to the user", func() {
			Expect(apiClient.UploadArtifacts(context.Background(), artifacts)).To(Succeed())
		})
	})
})
