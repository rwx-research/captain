package cli_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/rwx-research/captain-cli/internal/api"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

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
		core                zapcore.Core
		recordedLogs        *observer.ObservedLogs

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

		core, recordedLogs = observer.New(zapcore.InfoLevel)
		service = cli.Service{
			API: new(mocks.API),
			Log: zaptest.NewLogger(GinkgoT(), zaptest.WrapOptions(
				zap.WrapCore(func(original zapcore.Core) zapcore.Core { return core }),
			)).Sugar(),
			FileSystem: new(mocks.FileSystem),
			TaskRunner: new(mocks.TaskRunner),
			Parsers:    []parsing.Parser{new(mocks.Parser)},
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

		service.Parsers[0].(*mocks.Parser).MockParse = func(r io.Reader) (*v1.TestResults, error) {
			return &v1.TestResults{}, nil
		}

		// Caution: This needs to be an existing file. We don't actually read from it, however the glob expansion
		// functionality is not mocked, i.e. it still uses the file-system.
		testResultsFilePath = "../../go.mod"

		mockGlob := func(pattern string) ([]string, error) {
			return []string{testResultsFilePath}, nil
		}
		mockOpen := func(name string) (fs.File, error) {
			Expect(name).To(Equal(testResultsFilePath))
			fileOpened = true
			file := new(mocks.File)
			file.Reader = strings.NewReader("")
			return file, nil
		}
		service.FileSystem.(*mocks.FileSystem).MockOpen = mockOpen
		service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob

		arg = fmt.Sprintf("%d", GinkgoRandomSeed())
	})

	JustBeforeEach(func() {
		runConfig := cli.RunConfig{
			Args:                []string{arg},
			TestResultsFileGlob: testResultsFilePath,
			SuiteID:             "test",
		}

		err = service.RunSuite(ctx, runConfig)
	})

	Context("under expected conditions", func() {
		BeforeEach(func() {
			mockUploadTestResults := func(
				ctx context.Context,
				testSuite string,
				testResultsFiles []api.TestResultsFile,
			) ([]api.TestResultsUploadResult, error) {
				Expect(testResultsFiles).To(HaveLen(1))
				testResultsFileUploaded = true
				return []api.TestResultsUploadResult{{OriginalPaths: []string{testResultsFilePath}, Uploaded: true}}, nil
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

		It("logs the uploaded files", func() {
			logMessages := make([]string, 0)

			for _, log := range recordedLogs.All() {
				logMessages = append(logMessages, log.Message)
			}

			Expect(logMessages).To(ContainElement(ContainSubstring("Found 1 test result file")))
			Expect(logMessages).To(ContainElement(ContainSubstring(fmt.Sprintf("- Uploaded %v", testResultsFilePath))))
		})
	})

	Context("with an erroring command", func() {
		var (
			exitCode                    int
			firstFailedTestDescription  string
			secondFailedTestDescription string
		)

		BeforeEach(func() {
			exitCode = int(GinkgoRandomSeed() + 1)
			firstFailedTestDescription = fmt.Sprintf("first-description-%d", GinkgoRandomSeed()+2)
			secondFailedTestDescription = fmt.Sprintf("second-description-%d", GinkgoRandomSeed()+3)

			mockGetExitStatusFromError := func(error) (int, error) {
				return exitCode, nil
			}
			service.TaskRunner.(*mocks.TaskRunner).MockGetExitStatusFromError = mockGetExitStatusFromError

			service.Parsers[0].(*mocks.Parser).MockParse = func(r io.Reader) (*v1.TestResults, error) {
				return &v1.TestResults{
					Tests: []v1.Test{
						{
							Name:     "passed test",
							Location: &v1.Location{File: "/path/to/file.test"},
							Attempt: v1.TestAttempt{
								Status: v1.NewSuccessfulTestStatus(),
							},
						},
						{
							Name:     firstFailedTestDescription,
							Location: &v1.Location{File: "/path/to/file.test"},
							Attempt: v1.TestAttempt{
								Status: v1.NewFailedTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     secondFailedTestDescription,
							Location: &v1.Location{File: "/other/path/to/file.test"},
							Attempt: v1.TestAttempt{
								Status: v1.NewFailedTestStatus(nil, nil, nil),
							},
						},
					},
				}, nil
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

		Context("other unknown tests quarantined", func() {
			BeforeEach(func() {
				mockGetQuarantinedTestCases := func(
					ctx context.Context,
					testSuiteIdentifier string,
				) ([]api.QuarantinedTestCase, error) {
					return []api.QuarantinedTestCase{
						{
							CompositeIdentifier: fmt.Sprintf("%v -captain- %v", secondFailedTestDescription, "not/the/right/path.test"),
							IdentityComponents:  []string{"description", "file"},
							StrictIdentity:      true,
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetQuarantinedTestCases = mockGetQuarantinedTestCases
			})

			It("returns the error code of the command", func() {
				Expect(err).To(HaveOccurred())
				executionError, ok := errors.AsExecutionError(err)
				Expect(ok).To(BeTrue(), "Error is an execution error")
				Expect(executionError.Code).To(Equal(exitCode))
			})
		})

		Context("other unknown tests quarantined", func() {
			BeforeEach(func() {
				mockGetQuarantinedTestCases := func(
					ctx context.Context,
					testSuiteIdentifier string,
				) ([]api.QuarantinedTestCase, error) {
					return []api.QuarantinedTestCase{
						{
							CompositeIdentifier: "some-description -captain- huh",
							IdentityComponents:  []string{"description", "huh"},
							StrictIdentity:      true,
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetQuarantinedTestCases = mockGetQuarantinedTestCases
			})

			It("returns the error code of the command", func() {
				Expect(err).To(HaveOccurred())
				executionError, ok := errors.AsExecutionError(err)
				Expect(ok).To(BeTrue(), "Error is an execution error")
				Expect(executionError.Code).To(Equal(exitCode))
			})

			It("does not log any quarantined tests", func() {
				logMessages := make([]string, 0)

				for _, log := range recordedLogs.All() {
					logMessages = append(logMessages, log.Message)
				}

				Expect(logMessages).NotTo(ContainElement(ContainSubstring("under quarantine")))
				Expect(logMessages).NotTo(ContainElement(ContainSubstring("remaining actionable")))
				Expect(logMessages).NotTo(ContainElement(fmt.Sprintf("- %v", firstFailedTestDescription)))
				Expect(logMessages).NotTo(ContainElement(fmt.Sprintf("- %v", secondFailedTestDescription)))
			})
		})

		Context("some tests quarantined", func() {
			BeforeEach(func() {
				mockGetQuarantinedTestCases := func(
					ctx context.Context,
					testSuiteIdentifier string,
				) ([]api.QuarantinedTestCase, error) {
					return []api.QuarantinedTestCase{
						{
							CompositeIdentifier: fmt.Sprintf("%v -captain- %v", secondFailedTestDescription, "/other/path/to/file.test"),
							IdentityComponents:  []string{"description", "file"},
							StrictIdentity:      true,
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetQuarantinedTestCases = mockGetQuarantinedTestCases
			})

			It("returns the error code of the command", func() {
				Expect(err).To(HaveOccurred())
				executionError, ok := errors.AsExecutionError(err)
				Expect(ok).To(BeTrue(), "Error is an execution error")
				Expect(executionError.Code).To(Equal(exitCode))
			})

			It("logs the quarantined tests", func() {
				logMessages := make([]string, 0)

				for _, log := range recordedLogs.All() {
					logMessages = append(logMessages, log.Message)
				}

				Expect(logMessages).To(ContainElement(ContainSubstring("1 of 2 failures under quarantine")))
				Expect(logMessages).To(ContainElement(ContainSubstring("1 remaining actionable failure")))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- %v", firstFailedTestDescription)))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- %v", secondFailedTestDescription)))
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
							CompositeIdentifier: fmt.Sprintf("%v -captain- %v", firstFailedTestDescription, "/path/to/file.test"),
							IdentityComponents:  []string{"description", "file"},
							StrictIdentity:      true,
						},
						{
							CompositeIdentifier: fmt.Sprintf("%v -captain- %v", secondFailedTestDescription, "/other/path/to/file.test"),
							IdentityComponents:  []string{"description", "file"},
							StrictIdentity:      true,
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetQuarantinedTestCases = mockGetQuarantinedTestCases
			})

			It("doesn't return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("logs the quarantined tests", func() {
				logMessages := make([]string, 0)

				for _, log := range recordedLogs.All() {
					logMessages = append(logMessages, log.Message)
				}

				Expect(logMessages).To(ContainElement(ContainSubstring("2 of 2 failures under quarantine")))
				Expect(logMessages).NotTo(ContainElement(ContainSubstring("remaining actionable")))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- %v", firstFailedTestDescription)))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- %v", secondFailedTestDescription)))
			})
		})
	})
})
