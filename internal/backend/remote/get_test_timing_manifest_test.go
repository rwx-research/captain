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

var _ = Describe("GetTestTimingManifest", func() {
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
				Expect(req.URL.Path).To(HaveSuffix("/api/test_suites/timing_manifest"))
				Expect(req.URL.Query().Has("test_suite_identifier")).To(BeTrue())
				Expect(req.URL.Query().Has("commit_sha")).To(BeTrue())

				resp.Body = io.NopCloser(strings.NewReader(`
					{
						"file_timings": [
							{ "file_path": "some-file", "duration": 200 }
						]
					}
				`))
				resp.StatusCode = 200
				return &resp, nil
			}
		})

		It("returns the test file timing", func() {
			testFileTiming, err := apiClient.GetTestTimingManifest(context.Background(), "test-suite-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(testFileTiming).To(HaveLen(1))
		})
	})

	Context("when the response is successful", func() {
		BeforeEach(func() {
			host = "cloud.rwx.com"
			mockRoundTripper = func(req *http.Request) (*http.Response, error) {
				var resp http.Response

				Expect(req.Method).To(Equal(http.MethodGet))
				Expect(req.URL.Path).To(HaveSuffix("/captain/api/test_suites/timing_manifest"))
				Expect(req.URL.Query().Has("test_suite_identifier")).To(BeTrue())
				Expect(req.URL.Query().Has("commit_sha")).To(BeTrue())

				resp.Body = io.NopCloser(strings.NewReader(`
					{
						"file_timings": [
							{ "file_path": "some-file", "duration": 200 }
						]
					}
				`))
				resp.StatusCode = 200
				return &resp, nil
			}
		})

		It("returns the test file timing", func() {
			testFileTiming, err := apiClient.GetTestTimingManifest(context.Background(), "test-suite-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(testFileTiming).To(HaveLen(1))
		})
	})
})
