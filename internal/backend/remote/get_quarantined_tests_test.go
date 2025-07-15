package remote_test

import (
	"context"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/backend/remote"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetQuarantinedTests", func() {
	var (
		apiClient        remote.Client
		mockRoundTripper func(*http.Request) (*http.Response, error)
		host             string
	)

	JustBeforeEach(func() {
		apiClientConfig := remote.ClientConfig{Log: zap.NewNop().Sugar(), Host: host}
		apiClient = remote.Client{ClientConfig: apiClientConfig, RoundTrip: mockRoundTripper}
	})

	Context("when the response is successful against captain.build", func() {
		BeforeEach(func() {
			host = "captain.build"
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				Expect(req.Method).To(Equal(http.MethodGet))
				Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/quarantined_tests"))
				Expect(req.URL.Query().Has("test_suite_identifier")).To(BeTrue())

				resp.Body = io.NopCloser(strings.NewReader(`
					{
						"quarantined_tests": [
							{
								"composite_identifier": "q-1",
								"identity_components": ["component1", "component2"],
								"strict_identity": true
							},
							{
								"composite_identifier": "q-2",
								"identity_components": ["component3"],
								"strict_identity": false
							}
						]
					}
				`))
				resp.StatusCode = 200

				return &resp, nil
			}
		})

		It("returns the quarantined tests", func() {
			quarantinedTests, err := apiClient.GetQuarantinedTests(context.Background(), "test-suite-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(quarantinedTests).To(HaveLen(2))
			Expect(quarantinedTests[0].CompositeIdentifier).To(Equal("q-1"))
			Expect(quarantinedTests[0].IdentityComponents).To(Equal([]string{"component1", "component2"}))
			Expect(quarantinedTests[0].StrictIdentity).To(BeTrue())
			Expect(quarantinedTests[1].CompositeIdentifier).To(Equal("q-2"))
			Expect(quarantinedTests[1].IdentityComponents).To(Equal([]string{"component3"}))
			Expect(quarantinedTests[1].StrictIdentity).To(BeFalse())
		})
	})

	Context("when the response is successful against cloud.rwx.com", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				Expect(req.Method).To(Equal(http.MethodGet))
				Expect(req.URL.Path).To(HaveSuffix("/captain/api/test_suites/quarantined_tests"))
				Expect(req.URL.Query().Has("test_suite_identifier")).To(BeTrue())

				resp.Body = io.NopCloser(strings.NewReader(`
					{
						"quarantined_tests": [
							{
								"composite_identifier": "q-1",
								"identity_components": ["component1", "component2"],
								"strict_identity": true
							}
						]
					}
				`))
				resp.StatusCode = 200

				return &resp, nil
			}
		})

		It("returns the quarantined tests", func() {
			quarantinedTests, err := apiClient.GetQuarantinedTests(context.Background(), "test-suite-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(quarantinedTests).To(HaveLen(1))
			Expect(quarantinedTests[0].CompositeIdentifier).To(Equal("q-1"))
		})
	})

	Context("when the response is empty", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				Expect(req.Method).To(Equal(http.MethodGet))
				Expect(req.URL.Path).To(HaveSuffix("/captain/api/test_suites/quarantined_tests"))
				Expect(req.URL.Query().Has("test_suite_identifier")).To(BeTrue())

				resp.Body = io.NopCloser(strings.NewReader(`{ "quarantined_tests": [] }`))
				resp.StatusCode = 200

				return &resp, nil
			}
		})

		It("returns an empty list", func() {
			quarantinedTests, err := apiClient.GetQuarantinedTests(context.Background(), "test-suite-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(quarantinedTests).To(BeEmpty())
		})
	})

	Context("when the response is not successful", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				Expect(req.Method).To(Equal(http.MethodGet))
				Expect(req.URL.Path).To(HaveSuffix("quarantined_tests"))
				Expect(req.URL.Query().Has("test_suite_identifier")).To(BeTrue())

				resp.StatusCode = 400
				resp.Body = io.NopCloser(strings.NewReader(""))

				return &resp, nil
			}
		})

		It("returns an error", func() {
			quarantinedTests, err := apiClient.GetQuarantinedTests(context.Background(), "test-suite-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API backend encountered an error"))
			Expect(quarantinedTests).To(BeNil())
		})
	})

	Context("when the response body is invalid JSON", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(_ *http.Request) (*http.Response, error) {
				var resp http.Response

				resp.Body = io.NopCloser(strings.NewReader(`invalid json`))
				resp.StatusCode = 200
				resp.Header = http.Header{
					"Content-Type": []string{"application/json"},
				}

				return &resp, nil
			}
		})

		It("returns an error", func() {
			quarantinedTests, err := apiClient.GetQuarantinedTests(context.Background(), "test-suite-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to parse the response body"))
			Expect(quarantinedTests).To(BeNil())
		})
	})
})
