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
		arg                 string
		testResultsFilePath string
		err                 error
		ctx                 context.Context
		mockCommand         *mocks.Command
		service             cli.Service

		testResultsFileUploaded, commandStarted, commandFinished, fetchedQuarantinedTests, fileOpened bool
	)

	BeforeEach(func() {
		ctx = context.Background()
		err = nil
		testResultsFileUploaded = false
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

		service.Parsers[0].(*mocks.Parser).MockParse = func(r io.Reader) ([]testing.TestResult, error) {
			return []testing.TestResult{}, nil
		}

		// Caution: This needs to be an existing file. We don't actually read from it, however the glob expansion
		// functionality is not mocked, i.e. it still uses the file-system.
		testResultsFilePath = "../../go.mod"

		mockOpen := func(name string) (fs.File, error) {
			Expect(name).To(Equal(testResultsFilePath))
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
			Args:                []string{arg},
			TestResultsFileGlob: testResultsFilePath,
			SuiteName:           "test",
		}

		err = service.RunSuite(ctx, runConfig)
	})

	Context("under expected conditions", func() {
		BeforeEach(func() {
			mockUploadTestResults := func(ctx context.Context, testSuite string, testResultsFiles []api.TestResultsFile) error {
				Expect(testResultsFiles).To(HaveLen(1))
				testResultsFileUploaded = true
				return nil
			}
			service.API.(*mocks.API).MockUploadTestResults = mockUploadTestResults

			mockGetQuarantinedTestCases := func(
				ctx context.Context,
				testSuiteIdentifier string,
			) ([]api.QuarantinedTestCase, error) {
				fetchedQuarantinedTests = true
				return nil, nil
			}
			service.API.(*mocks.API).MockGetQuarantinedTestCases = mockGetQuarantinedTestCases

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

		It("reads the test results file", func() {
			Expect(fileOpened).To(BeTrue())
		})

		It("uploads the test results file", func() {
			Expect(testResultsFileUploaded).To(BeTrue())
		})
	})

	Context("with an erroring command", func() {
		var (
			exitCode              int
			failedTestDescription string
		)

		BeforeEach(func() {
			exitCode = int(GinkgoRandomSeed() + 1)
			failedTestDescription = fmt.Sprintf("%d", GinkgoRandomSeed()+2)

			mockGetExitStatusFromError := func(error) (int, error) {
				return exitCode, nil
			}
			service.TaskRunner.(*mocks.TaskRunner).MockGetExitStatusFromError = mockGetExitStatusFromError

			service.Parsers[0].(*mocks.Parser).MockParse = func(r io.Reader) ([]testing.TestResult, error) {
				return []testing.TestResult{{Description: failedTestDescription, Status: testing.TestStatusFailed}}, nil
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
				mockGetQuarantinedTestCases := func(
					ctx context.Context,
					testSuiteIdentifier string,
				) ([]api.QuarantinedTestCase, error) {
					return []api.QuarantinedTestCase{
						{
							CompositeIdentifier: failedTestDescription,
							IdentityComponents:  []string{"description"},
							StrictIdentity:      true,
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetQuarantinedTestCases = mockGetQuarantinedTestCases
			})

			It("doesn't return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
