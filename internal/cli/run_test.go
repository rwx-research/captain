package cli_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap/zaptest"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Run", func() {
	var (
		arg          string
		artifactPath string
		err          error
		ctx          context.Context
		mockCommand  *mocks.Command
		service      cli.Service

		artifactUploaded, commandStarted, commandFinished, fetchedQuarantinedTests, fileOpened bool
	)

	BeforeEach(func() {
		ctx = context.Background()
		err = nil
		artifactUploaded = false
		commandStarted = false
		commandFinished = false
		fetchedQuarantinedTests = false
		fileOpened = false

		service = cli.Service{
			API:        new(mocks.API),
			Log:        zaptest.NewLogger(GinkgoT()).Sugar(),
			FileSystem: new(mocks.FileSystem),
			TaskRunner: new(mocks.TaskRunner),
			Parsers:    []cli.Parser{new(mocks.Parser)},
		}

		mockCommand = new(mocks.Command)
		mockCommand.MockStart = func() error {
			commandStarted = true
			return nil
		}

		newCommand := func(ctx context.Context, name string, args ...string) (exec.Command, error) {
			Expect(args).To(HaveLen(0))
			Expect(name).To(Equal(arg))
			return mockCommand, nil
		}
		service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand

		service.Parsers[0].(*mocks.Parser).MockParse = func(r io.Reader) (map[string]testing.TestResult, error) {
			return make(map[string]testing.TestResult), nil
		}

		// Caution: This needs to be an existing file. We don't actually read from it, however the glob expansion
		// functionality is not mocked, i.e. it still uses the file-system.
		artifactPath = "../../go.mod"

		mockOpen := func(name string) (fs.File, error) {
			Expect(name).To(Equal(artifactPath))
			fileOpened = true
			file := new(mocks.File)
			file.Reader = strings.NewReader("")
			return file, nil
		}
		service.FileSystem.(*mocks.FileSystem).MockOpen = mockOpen

		arg = fmt.Sprintf("%d", GinkgoRandomSeed())
	})

	JustBeforeEach(func() {
		runConfig := cli.RunConfig{
			Args:         []string{arg},
			ArtifactGlob: artifactPath,
			SuiteName:    "test",
		}

		err = service.RunSuite(ctx, runConfig)
	})

	Context("under expected conditions", func() {
		BeforeEach(func() {
			mockUploadArtifacts := func(ctx context.Context, artifacts []api.Artifact) error {
				Expect(artifacts).To(HaveLen(1))
				artifactUploaded = true
				return nil
			}
			service.API.(*mocks.API).MockUploadArtifacts = mockUploadArtifacts

			mockGetQuarantinedTestIDs := func(ctx context.Context) ([]string, error) {
				fetchedQuarantinedTests = true
				return nil, nil
			}
			service.API.(*mocks.API).MockGetQuarantinedTestIDs = mockGetQuarantinedTestIDs

			mockCommand.MockWait = func() error {
				commandFinished = true
				return nil
			}
		})

		It("doesn't return an error", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("runs the supplied command", func() {
			Expect(commandStarted).To(BeTrue())
		})

		It("waits until the supplied command stopped running", func() {
			Expect(commandFinished).To(BeTrue())
		})

		It("fetches a list of quarantined tests", func() {
			Expect(fetchedQuarantinedTests).To(BeTrue())
		})

		It("reads the artifact file", func() {
			Expect(fileOpened).To(BeTrue())
		})

		It("uploads the artifact", func() {
			Expect(artifactUploaded).To(BeTrue())
		})
	})

	Context("with an erroring command", func() {
		var (
			exitCode     int
			failedTestID string
		)

		BeforeEach(func() {
			exitCode = int(GinkgoRandomSeed() + 1)
			failedTestID = fmt.Sprintf("%d", GinkgoRandomSeed()+2)

			mockGetExitStatusFromError := func(error) (int, error) {
				return exitCode, nil
			}
			service.TaskRunner.(*mocks.TaskRunner).MockGetExitStatusFromError = mockGetExitStatusFromError

			service.Parsers[0].(*mocks.Parser).MockParse = func(r io.Reader) (map[string]testing.TestResult, error) {
				result := make(map[string]testing.TestResult)
				result[failedTestID] = testing.TestResult{Status: testing.TestStatusFailed}

				return result, nil
			}
		})

		Context("no quarantined tests", func() {
			It("returns the error code of the command", func() {
				Expect(err).To(HaveOccurred())
				executionError, ok := errors.AsExecutionError(err)
				Expect(ok).To(BeTrue(), "Error is an execution error")
				Expect(executionError.Code).To(Equal(exitCode))
			})
		})

		Context("all tests quarantined", func() {
			BeforeEach(func() {
				mockGetQuarantinedTestIDs := func(ctx context.Context) ([]string, error) {
					return []string{failedTestID}, nil
				}
				service.API.(*mocks.API).MockGetQuarantinedTestIDs = mockGetQuarantinedTestIDs
			})

			It("doesn't return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
