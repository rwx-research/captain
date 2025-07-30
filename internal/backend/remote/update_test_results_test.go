package remote_test

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/backend/remote"
	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Uploading Test Results", func() {
	var (
		apiClient          remote.Client
		testResults        v1.TestResults
		mockNewUUID        func() (uuid.UUID, error)
		mockRoundTripper   func(*http.Request) (*http.Response, error)
		mockRoundTripCalls int
		mockUUID           uuid.UUID
		host               string
	)

	BeforeEach(func() {
		testResults = v1.TestResults{}
		mockUUID = uuid.MustParse("fff24366-af1d-43cc-ab32-8c9ed137cf09")
		mockNewUUID = func() (uuid.UUID, error) {
			return mockUUID, nil
		}

		mockRoundTripCalls = 0
	})

	JustBeforeEach(func() {
		apiClient = remote.Client{ClientConfig: remote.ClientConfig{
			Log:     zap.NewNop().Sugar(),
			NewUUID: mockNewUUID,
			Host:    host,
		}, RoundTrip: mockRoundTripper}
	})

	Context("under expected conditions w/ captain.build host", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the test results file
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/bulk_test_results"))
					Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", mockUUID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.String()).To(ContainSubstring(fmt.Sprintf("%d", GinkgoRandomSeed())))
					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/bulk_test_results"))
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
			uploadResults, err := apiClient.UpdateTestResults(context.Background(), "test suite id", testResults)
			Expect(err).To(Succeed())
			Expect(uploadResults).To(HaveLen(1))
			Expect(uploadResults[0].Uploaded).To(Equal(true))
		})
	})

	Context("under expected conditions", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the test results file
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(HaveSuffix("/captain/api/test_suites/bulk_test_results"))
					Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", mockUUID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.String()).To(ContainSubstring(fmt.Sprintf("%d", GinkgoRandomSeed())))
					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.Path).To(HaveSuffix("/captain/api/test_suites/bulk_test_results"))
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
			uploadResults, err := apiClient.UpdateTestResults(context.Background(), "test suite id", testResults)
			Expect(err).To(Succeed())
			Expect(uploadResults).To(HaveLen(1))
			Expect(uploadResults[0].Uploaded).To(Equal(true))
		})
	})

	Context("with large contents in `derivedFrom`", func() {
		var err error

		BeforeEach(func() {
			err = nil
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the test results file
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/bulk_test_results"))
					Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", mockUUID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.String()).To(ContainSubstring(fmt.Sprintf("%d", GinkgoRandomSeed())))

					body, err := io.ReadAll(req.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(ContainSubstring("\"contents\":\"PHRydW5jYXRlZCBkdWUgdG8gdGVzdCByZXN1bHRzIHNpemU+\""))

					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/bulk_test_results"))
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

		JustBeforeEach(func() {
			tooMuchData := make([]byte, 26*1024*1024)
			_, _ = rand.NewChaCha8([32]byte{}).Read(tooMuchData)

			testResults = v1.TestResults{
				DerivedFrom: []v1.OriginalTestResults{{
					Contents: string(tooMuchData),
				}},
			}
			_, err = apiClient.UpdateTestResults(context.Background(), "test suite id", testResults)
		})

		It("strips the `derivedFrom` content from the test results", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with large backtraces in previous attempts", func() {
		var err error

		BeforeEach(func() {
			err = nil
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the test results file
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/bulk_test_results"))
					Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", mockUUID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.String()).To(ContainSubstring(fmt.Sprintf("%d", GinkgoRandomSeed())))

					body, err := io.ReadAll(req.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(ContainSubstring("\"contents\":\"PHRydW5jYXRlZCBkdWUgdG8gdGVzdCByZXN1bHRzIHNpemU+\""))
					Expect(string(body)).To(ContainSubstring("\"backtrace\":[\"\\u003ctruncated due to test results size\\u003e\"]"))
					Expect(string(body)).To(ContainSubstring("\"backtrace\":[\"won't be touched\"]"))

					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/bulk_test_results"))
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

		JustBeforeEach(func() {
			tooMuchData := make([]byte, 26*1024*1024)
			_, _ = rand.NewChaCha8([32]byte{}).Read(tooMuchData)

			testResults = v1.TestResults{
				Tests: []v1.Test{{
					PastAttempts: []v1.TestAttempt{{
						Status: v1.TestStatus{
							Backtrace: []string{string(tooMuchData)},
						},
					}},
					Attempt: v1.TestAttempt{
						Status: v1.TestStatus{
							Backtrace: []string{"won't be touched"},
						},
					},
				}},
				DerivedFrom: []v1.OriginalTestResults{{
					Contents: "will be stripped regardless",
				}},
			}
			_, err = apiClient.UpdateTestResults(context.Background(), "test suite id", testResults)
		})

		It("strips the `derivedFrom` content from the test results", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with large backtraces in current attempt", func() {
		var err error

		BeforeEach(func() {
			err = nil
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the test results file
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/bulk_test_results"))
					Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", mockUUID, GinkgoRandomSeed(),
					)))
				case 1: // upload to `upload_url`
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.String()).To(ContainSubstring(fmt.Sprintf("%d", GinkgoRandomSeed())))

					body, err := io.ReadAll(req.Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(ContainSubstring("\"contents\":\"PHRydW5jYXRlZCBkdWUgdG8gdGVzdCByZXN1bHRzIHNpemU+\""))
					Expect(string(body)).To(ContainSubstring("\"backtrace\":[\"\\u003ctruncated due to test results size\\u003e\"]"))
					Expect(string(body)).NotTo(ContainSubstring("\"backtrace\":[\"will be stripped regardless\"]"))

					resp.Body = io.NopCloser(strings.NewReader(""))
				case 2: // update status
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/bulk_test_results"))
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

		JustBeforeEach(func() {
			tooMuchData := make([]byte, 26*1024*1024)
			_, _ = rand.NewChaCha8([32]byte{}).Read(tooMuchData)

			testResults = v1.TestResults{
				Tests: []v1.Test{{
					PastAttempts: []v1.TestAttempt{{
						Status: v1.TestStatus{
							Backtrace: []string{"will be stripped regardless"},
						},
					}},
					Attempt: v1.TestAttempt{
						Status: v1.TestStatus{
							Backtrace: []string{string(tooMuchData)},
						},
					},
				}},
				DerivedFrom: []v1.OriginalTestResults{{
					Contents: "will be stripped regardless",
				}},
			}
			_, err = apiClient.UpdateTestResults(context.Background(), "test suite id", testResults)
		})

		It("strips the `derivedFrom` content from the test results", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with an error during test results file registration", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(_ *http.Request) (*http.Response, error) {
				return nil, errors.NewInternalError("Error")
			}
		})

		It("returns an error to the user", func() {
			uploadResults, err := apiClient.UpdateTestResults(context.Background(), "test suite id", testResults)
			Expect(uploadResults).To(BeNil())
			Expect(err).ToNot(Succeed())
		})
	})

	Context("with an error from S3", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				switch mockRoundTripCalls {
				case 0: // registering the test results file
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", mockUUID, GinkgoRandomSeed(),
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
			uploadResults, err := apiClient.UpdateTestResults(context.Background(), "test suite id", testResults)
			Expect(err).To(Succeed())
			Expect(uploadResults).To(HaveLen(1))
			Expect(uploadResults[0].Uploaded).To(Equal(false))
		})
	})

	Context("with an error while updating an test results file status", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(_ *http.Request) (*http.Response, error) {
				var (
					resp http.Response
					err  error
				)

				switch mockRoundTripCalls {
				case 0: // registering the test results files
					resp.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(
						"{\"test_results_uploads\":[{\"id\": %q, \"external_identifier\":%q,\"upload_url\":\"%d\"}]}",
						"some-captain-identifier", mockUUID, GinkgoRandomSeed(),
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
			uploadResults, err := apiClient.UpdateTestResults(context.Background(), "test suite id", testResults)
			Expect(err).To(Succeed())
			Expect(uploadResults).To(HaveLen(1))
			Expect(uploadResults[0].Uploaded).To(Equal(true))
		})
	})
})
