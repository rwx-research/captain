package local_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/backend/local"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mocks"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("local backend client", func() {
	It("persists opaque timings separately by granularity without overwriting legacy files", func() {
		dir := GinkgoT().TempDir()
		newClient := func() local.Client {
			client, err := local.NewClient(fs.Local{}, filepath.Join(dir, "flakes"),
				filepath.Join(dir, "quarantines"), filepath.Join(dir, "timings"))
			Expect(err).NotTo(HaveOccurred())
			return client
		}
		client := newClient()
		ctx := context.Background()
		for _, manifest := range []v1.TimingManifest{
			{Granularity: "file", Timings: []v1.Timing{{Identifier: "same", Duration: time.Second}}},
			{Granularity: "package", Timings: []v1.Timing{{Identifier: "same", Duration: 2 * time.Second}}},
			{Granularity: "custom/../kind", Timings: []v1.Timing{{Identifier: "same", Duration: 3 * time.Second}}},
		} {
			_, err := client.UpdateTestResults(ctx, "suite", v1.TestResults{TimingManifests: []v1.TimingManifest{manifest}})
			Expect(err).NotTo(HaveOccurred())
		}
		client = newClient()
		for i, granularity := range []string{"file", "package", "custom/../kind"} {
			timings, err := client.GetTestTimingManifest(ctx, "suite", granularity)
			Expect(err).NotTo(HaveOccurred())
			Expect(timings).To(HaveLen(1))
			Expect(timings[0].Filepath).To(Equal("same"))
			Expect(timings[0].Duration).To(Equal(time.Duration(i+1) * time.Second))
		}
	})

	const (
		flakesPath      = "flakes.yaml"
		quarantinesPath = "quarantines.yaml"
		timingsPath     = "timings.yaml"
	)

	var (
		err                          error
		client                       local.Client
		fileSystem                   mocks.FileSystem
		flakes, quarantines, timings mocks.File
	)

	BeforeEach(func() {
		flakes.Reader = strings.NewReader("")
		quarantines.Reader = strings.NewReader("")
		timings.Reader = strings.NewReader("")

		fileSystem.MockOpen = func(name string) (fs.File, error) {
			switch name {
			case flakesPath:
				return &flakes, nil
			case quarantinesPath:
				return &quarantines, nil
			case timingsPath:
				return &timings, nil
			default:
				return nil, os.ErrNotExist
			}
		}

		client, err = local.NewClient(&fileSystem, flakesPath, quarantinesPath, timingsPath)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("UploadTestResults", func() {
		const suiteID = "suite-id"

		var (
			duration      time.Duration
			testResults   v1.TestResults
			uploadResults []backend.TestResultsUploadResult
		)

		BeforeEach(func() {
			duration = time.Second * time.Duration(GinkgoRandomSeed())
			flakes.Builder = new(strings.Builder)
			quarantines.Builder = new(strings.Builder)
			timings.Builder = new(strings.Builder)

			fileSystem.MockOpenFile = func(name string, _ int, _ os.FileMode) (fs.File, error) {
				switch name {
				case flakesPath:
					return &flakes, nil
				case quarantinesPath:
					return &quarantines, nil
				case timingsPath:
					return &timings, nil
				default:
					return nil, os.ErrNotExist
				}
			}

			testResults = v1.TestResults{
				Framework: v1.Framework{
					Kind:     v1.FrameworkKindCypress,
					Language: v1.FrameworkLanguageJavaScript,
				},
				Tests: []v1.Test{{
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Status: v1.TestStatus{
							Kind: v1.TestStatusSuccessful,
						},
					},
					Location: &v1.Location{
						File: fmt.Sprintf("%d", GinkgoRandomSeed()),
					},
				}},
			}
		})

		JustBeforeEach(func() {
			uploadResults, err = client.UpdateTestResults(context.Background(), suiteID, testResults)
		})

		It("updates the timings file", func() {
			var result map[string]time.Duration

			Expect(err).ToNot(HaveOccurred())
			Expect(uploadResults).To(HaveLen(1))
			Expect(yaml.Unmarshal([]byte(timings.Builder.String()), &result)).To(Succeed())
			Expect(result).To(HaveKey(fmt.Sprintf("%d", GinkgoRandomSeed())))
			Expect(result[fmt.Sprintf("%d", GinkgoRandomSeed())]).To(Equal(time.Second * time.Duration(GinkgoRandomSeed())))
		})
	})

	Describe("GetQuarantinedTests", func() {
		var (
			quarantinedTests []backend.Test
			quarantineTime   time.Time
		)

		Context("when there are quarantined tests", func() {
			BeforeEach(func() {
				quarantineTime = time.Now().Truncate(time.Second)
				quarantines.Reader = strings.NewReader(`- name: Test 1
  file: test1.js
  strict: "true"
- name: Test 2
  file: test2.js
  strict: "false"`)

				quarantines.MockModTime = func() time.Time {
					return quarantineTime
				}

				fileSystem.MockOpen = func(name string) (fs.File, error) {
					switch name {
					case flakesPath:
						return &flakes, nil
					case quarantinesPath:
						return &quarantines, nil
					case timingsPath:
						return &timings, nil
					default:
						return nil, os.ErrNotExist
					}
				}

				client, err = local.NewClient(&fileSystem, flakesPath, quarantinesPath, timingsPath)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the quarantined tests", func() {
				quarantinedTests, err = client.GetQuarantinedTests(context.Background(), "test-suite-id")
				Expect(err).ToNot(HaveOccurred())
				Expect(quarantinedTests).To(HaveLen(2))

				Expect(quarantinedTests[0].CompositeIdentifier).To(Equal("Test 1 -captain- test1.js"))
				Expect(quarantinedTests[0].IdentityComponents).To(Equal([]string{"name", "file"}))
				Expect(quarantinedTests[0].StrictIdentity).To(BeTrue())

				Expect(quarantinedTests[1].CompositeIdentifier).To(Equal("Test 2 -captain- test2.js"))
				Expect(quarantinedTests[1].IdentityComponents).To(Equal([]string{"name", "file"}))
				Expect(quarantinedTests[1].StrictIdentity).To(BeFalse())
			})
		})

		Context("when there are no quarantined tests", func() {
			BeforeEach(func() {
				quarantineTime = time.Now().Truncate(time.Second)
				quarantines.Reader = strings.NewReader("")

				quarantines.MockModTime = func() time.Time {
					return quarantineTime
				}

				fileSystem.MockOpen = func(name string) (fs.File, error) {
					switch name {
					case flakesPath:
						return &flakes, nil
					case quarantinesPath:
						return &quarantines, nil
					case timingsPath:
						return &timings, nil
					default:
						return nil, os.ErrNotExist
					}
				}

				client, err = local.NewClient(&fileSystem, flakesPath, quarantinesPath, timingsPath)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an empty list", func() {
				quarantinedTests, err = client.GetQuarantinedTests(context.Background(), "test-suite-id")
				Expect(err).ToNot(HaveOccurred())
				Expect(quarantinedTests).To(BeEmpty())
			})
		})
	})
})
