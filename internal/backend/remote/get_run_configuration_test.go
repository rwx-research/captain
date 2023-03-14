package remote_test

import (
	"context"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/backend/remote"
	"github.com/rwx-research/captain-cli/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetRunConfiguration", func() {
	var (
		apiClient        remote.Client
		mockRoundTripper func(*http.Request) (*http.Response, error)
	)

	JustBeforeEach(func() {
		apiClientConfig := remote.ClientConfig{Log: zap.NewNop().Sugar(), Provider: providers.GithubProvider{}}
		apiClient = remote.Client{ClientConfig: apiClientConfig, RoundTrip: mockRoundTripper}
	})

	Context("when the response is successful", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				Expect(req.Method).To(Equal(http.MethodGet))
				Expect(req.URL.Path).To(HaveSuffix("run_configuration"))
				Expect(req.URL.Query().Has("test_suite_identifier")).To(BeTrue())

				resp.Body = io.NopCloser(strings.NewReader(`
					{
						"generated_at": "some-time",
						"quarantined_tests": [{"composite_identifier": "q-1"}],
						"flaky_tests": [{"composite_identifier": "f-1"}]
					}
				`))
				resp.StatusCode = 200

				return &resp, nil
			}
		})

		It("returns the run configuration", func() {
			runConfiguration, err := apiClient.GetRunConfiguration(context.Background(), "test-suite-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(runConfiguration.QuarantinedTests).To(HaveLen(1))
			Expect(runConfiguration.QuarantinedTests[0].CompositeIdentifier).To(Equal("q-1"))
			Expect(runConfiguration.FlakyTests).To(HaveLen(1))
			Expect(runConfiguration.FlakyTests[0].CompositeIdentifier).To(Equal("f-1"))
		})
	})

	Context("when the response is not successful", func() {
		BeforeEach(func() {
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				Expect(req.Method).To(Equal(http.MethodGet))
				Expect(req.URL.Path).To(HaveSuffix("run_configuration"))
				Expect(req.URL.Query().Has("test_suite_identifier")).To(BeTrue())

				resp.StatusCode = 400
				resp.Body = io.NopCloser(strings.NewReader(""))

				return &resp, nil
			}
		})

		It("returns an error", func() {
			runConfiguration, err := apiClient.GetRunConfiguration(context.Background(), "test-suite-id")
			Expect(err).To(HaveOccurred())
			Expect(runConfiguration.QuarantinedTests).To(BeNil())
			Expect(runConfiguration.FlakyTests).To(BeNil())
		})
	})
})
