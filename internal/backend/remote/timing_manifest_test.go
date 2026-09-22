package remote_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/backend/remote"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("entity timing manifests", func() {
	It("encodes an empty manifest as an array rather than null", func() {
		client := remote.Client{ClientConfig: remote.ClientConfig{Host: "cloud.rwx.com"}}
		client.RoundTrip = func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(MatchJSON(`{"test_suite_identifier":"suite","granularity":"package","timings":[]}`))
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		Expect(client.UploadTimingManifest(context.Background(), "suite", v1.TimingManifest{
			Granularity: "package",
		})).To(Succeed())
	})

	It("selects arbitrary read granularity and uses identifiers rather than legacy file timings", func() {
		client := remote.Client{ClientConfig: remote.ClientConfig{Host: "cloud.rwx.com"}}
		client.RoundTrip = func(req *http.Request) (*http.Response, error) {
			Expect(req.URL.Query().Get("granularity")).To(Equal("test-case"))
			Expect(req.URL.Query().Get("test_suite_identifier")).To(Equal("suite"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{
				"file_timings":[{"file_path":"wrong","duration_in_nanoseconds":99}],
				"granularity":"test-case","timings":[{"identifier":"opaque/../case","duration_in_nanoseconds":123}]
			}`))}, nil
		}
		timings, err := client.GetTestTimingManifest(context.Background(), "suite", "test-case")
		Expect(err).NotTo(HaveOccurred())
		Expect(timings).To(HaveLen(1))
		Expect(timings[0].Filepath).To(Equal("opaque/../case"))
		Expect(timings[0].Duration).To(Equal(123 * time.Nanosecond))
	})

	DescribeTable("suppresses extraction then uploads the independent manifest and propagates failure",
		func(status int) {
			id := uuid.New()
			client := remote.Client{ClientConfig: remote.ClientConfig{
				Host: "cloud.rwx.com", Log: zap.NewNop().Sugar(), NewUUID: func() (uuid.UUID, error) { return id, nil },
			}}
			calls := 0
			client.RoundTrip = func(req *http.Request) (*http.Response, error) {
				calls++
				body, err := io.ReadAll(req.Body)
				Expect(err).NotTo(HaveOccurred())
				responseBody := ""
				responseStatus := http.StatusOK
				switch calls {
				case 1:
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(Equal("/captain/api/test_suites/bulk_test_results"))
					var registration struct {
						Files []struct {
							Skip bool `json:"skip_file_timings"`
						} `json:"test_results_files"`
					}
					Expect(json.Unmarshal(body, &registration)).To(Succeed())
					Expect(registration.Files).To(HaveLen(1))
					Expect(registration.Files[0].Skip).To(BeTrue())
					responseBody = fmt.Sprintf(`{"test_results_uploads":[{
						"id":"upload","external_identifier":%q,"upload_url":"https://example.com/results"}]}`, id)
				case 2:
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(req.URL.String()).To(Equal("https://example.com/results"))
					Expect(string(body)).NotTo(ContainSubstring("granularity"))
				case 3:
					Expect(req.Method).To(Equal(http.MethodPut))
					Expect(string(body)).To(ContainSubstring(`"upload_status":"uploaded"`))
				case 4:
					Expect(req.Method).To(Equal(http.MethodPost))
					Expect(req.URL.Path).To(Equal("/captain/api/test_suites/timings"))
					Expect(string(body)).To(MatchJSON(`{"test_suite_identifier":"suite","granularity":"package",
						"timings":[{"identifier":"example/pkg","duration_in_nanoseconds":1250000000}]}`))
					responseStatus = status
				default:
					Fail("unexpected request")
				}
				return &http.Response{StatusCode: responseStatus, Body: io.NopCloser(strings.NewReader(responseBody))}, nil
			}
			results := *v1.NewTestResults(v1.GoTestFramework, nil, nil)
			results.TimingManifests = []v1.TimingManifest{{Granularity: "package", Timings: []v1.Timing{
				{Identifier: "example/pkg", Duration: 1250 * time.Millisecond},
			}}}
			_, err := client.UpdateTestResults(context.Background(), "suite", results)
			if status == http.StatusCreated {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
			Expect(calls).To(Equal(4))
		}, Entry("201", http.StatusCreated), Entry("failed upload", http.StatusUnprocessableEntity))
})
