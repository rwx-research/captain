package cli_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
		filesOpened         []string
		mockAbqCommand      *mocks.Command
		mockAbqCommandArgs  []string
		abqStateJSON        string
		abqExecutable       string

		testResultsFileUploaded, commandStarted, commandFinished, fetchedQuarantinedTests bool
		abqStateFileExists, abqCommandStarted, abqCommandFinished                         bool
	)

	BeforeEach(func() {
		ctx = context.Background()
		err = nil
		testResultsFileUploaded = false
		commandStarted = false
		commandFinished = false
		fetchedQuarantinedTests = false
		filesOpened = make([]string, 0)
		mockAbqCommandArgs = make([]string, 0)
		abqExecutable = "/path/to/abq"
		abqStateJSON = ""
		abqStateFileExists = false
		abqCommandStarted = false
		abqCommandFinished = false

		core, recordedLogs = observer.New(zapcore.InfoLevel)
		log := zaptest.NewLogger(GinkgoT(), zaptest.WrapOptions(
			zap.WrapCore(func(original zapcore.Core) zapcore.Core { return core }),
		)).Sugar()
		service = cli.Service{
			API:        new(mocks.API),
			Log:        log,
			FileSystem: new(mocks.FileSystem),
			TaskRunner: new(mocks.TaskRunner),
			ParseConfig: parsing.Config{
				MutuallyExclusiveParsers: []parsing.Parser{new(mocks.Parser)},
				Logger:                   log,
			},
		}

		mockCommand = new(mocks.Command)
		mockCommand.MockStart = func() error {
			commandStarted = true
			return nil
		}

		mockAbqCommand = new(mocks.Command)
		mockAbqCommand.MockStart = func() error {
			abqCommandStarted = true
			return nil
		}
		mockAbqCommand.MockWait = func() error {
			abqCommandFinished = true
			return nil
		}

		newCommand := func(ctx context.Context, name string, args []string, environ []string) (exec.Command, error) {
			switch name {
			case abqExecutable:
				mockAbqCommandArgs = args
				Expect(environ).To(HaveLen(0))
				return mockAbqCommand, nil
			default:
				Expect(args).To(HaveLen(0))
				Expect(name).To(Equal(arg))
				Expect(environ).To(HaveLen(2))
				Expect(environ).To(ContainElement("ABQ_SET_EXIT_CODE=false"))
				Expect(environ).To(ContainElement(ContainSubstring("ABQ_STATE_FILE=tmp/captain-abq-")))
				return mockCommand, nil
			}
		}
		service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand

		service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(r io.Reader) (
			*v1.TestResults,
			error,
		) {
			return &v1.TestResults{}, nil
		}

		// Caution: This needs to be an existing file. We don't actually read from it, however the glob expansion
		// functionality is not mocked, i.e. it still uses the file-system.
		testResultsFilePath = "../../go.mod"

		mockGlob := func(pattern string) ([]string, error) {
			return []string{testResultsFilePath}, nil
		}
		mockOpen := func(name string) (fs.File, error) {
			const abqStateSubstring string = "tmp/captain-abq-"

			Expect(name).To(SatisfyAny(
				Equal(testResultsFilePath),
				ContainSubstring(abqStateSubstring),
			))
			filesOpened = append(filesOpened, name)

			isAbqStateFile := strings.Contains(name, abqStateSubstring)
			file := new(mocks.File)

			if isAbqStateFile {
				if !abqStateFileExists {
					return nil, os.ErrNotExist
				}

				file.Reader = strings.NewReader(abqStateJSON)
			} else {
				file.Reader = strings.NewReader("")
			}
			return file, nil
		}
		service.FileSystem.(*mocks.FileSystem).MockOpen = mockOpen
		service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob

		arg = fmt.Sprintf("fake-command-%d", GinkgoRandomSeed())
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
			Expect(filesOpened).To(ContainElement(testResultsFilePath))
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

		Context("ABQ", func() {
			It("attempts to read the ABQ state file", func() {
				Expect(filesOpened).To(ContainElement(ContainSubstring("tmp/captain-abq-")))
			})

			Context("when state file doesn't exist", func() {
				BeforeEach(func() {
					abqStateFileExists = false
				})

				It("does not call abq", func() {
					Expect(abqCommandStarted).To(BeFalse())
					Expect(abqCommandFinished).To(BeFalse())
				})
			})

			Context("with a supervisor ABQ state file", func() {
				BeforeEach(func() {
					abqStateFileExists = true
					abqStateJSON = fmt.Sprintf(`{
          "abq_version": "1.1.2",
          "abq_executable": "%s",
          "run_id": "not-a-real-run-id",
          "supervisor": true
        }`, abqExecutable)
				})

				It("calls abq set-exit-code", func() {
					Expect(abqCommandStarted).To(BeTrue())
					Expect(abqCommandFinished).To(BeTrue())
					Expect(mockAbqCommandArgs).To(Equal([]string{
						"set-exit-code",
						"--run-id", "not-a-real-run-id",
						"--exit-code", "0",
					}))
					Expect(err).NotTo(HaveOccurred())
				})

				Context("when `abq set-exit-code` fails to start", func() {
					BeforeEach(func() {
						mockAbqCommand.MockStart = func() error {
							abqCommandStarted = true
							return errors.NewSystemError("abq set-exit-code failed to start (forced)")
						}
					})

					It("logs an error", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeFalse())
						Expect(err).To(HaveOccurred())
						_, ok := errors.AsSystemError(err)
						Expect(ok).To(BeTrue(), "Error is a system error")

						logMessages := make([]string, 0)

						for _, log := range recordedLogs.All() {
							logMessages = append(logMessages, log.Message)
						}

						Expect(logMessages).To(ContainElement(ContainSubstring(
							"Error setting ABQ exit code: unable to execute sub-command: abq set-exit-code failed to start (forced)",
						)))
					})
				})

				Context("when `abq set-exit-code` returns an error", func() {
					BeforeEach(func() {
						mockAbqCommand.MockWait = func() error {
							abqCommandFinished = true
							return errors.NewExecutionError(1, "abq set-exit-code returned an error (forced)")
						}
					})

					It("logs an error", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeTrue())
						Expect(err).To(HaveOccurred())

						logMessages := make([]string, 0)

						for _, log := range recordedLogs.All() {
							logMessages = append(logMessages, log.Message)
						}

						Expect(logMessages).To(ContainElement(ContainSubstring(
							"Error setting ABQ exit code: Error during program execution: abq set-exit-code returned an error (forced)",
						)))
					})
				})
			})

			Context("with a non-supervisor ABQ state file", func() {
				BeforeEach(func() {
					abqStateFileExists = true
					abqStateJSON = fmt.Sprintf(`{
          "abq_version": "1.1.2",
          "abq_executable": "%s",
          "run_id": "not-a-real-run-id",
          "supervisor": false
        }`, abqExecutable)
				})

				It("does not call abq", func() {
					Expect(abqCommandStarted).To(BeFalse())
					Expect(abqCommandFinished).To(BeFalse())
				})
			})

			Context("with a empty, existent ABQ state file", func() {
				BeforeEach(func() {
					abqStateFileExists = true
					abqStateJSON = ""
				})

				It("does not call abq", func() {
					Expect(abqCommandStarted).To(BeFalse())
					Expect(abqCommandFinished).To(BeFalse())
				})

				It("logs an ABQ error", func() {
					logMessages := make([]string, 0)

					for _, log := range recordedLogs.All() {
						logMessages = append(logMessages, log.Message)
					}

					Expect(logMessages).To(ContainElement(ContainSubstring("Error decoding ABQ state file")))
				})

				Context("with a non-existent ABQ state file", func() {
					BeforeEach(func() {
						abqStateFileExists = false
					})

					It("does not call abq", func() {
						Expect(abqCommandStarted).To(BeFalse())
						Expect(abqCommandFinished).To(BeFalse())
					})

					It("does not log an ABQ error", func() {
						logMessages := make([]string, 0)

						for _, log := range recordedLogs.All() {
							logMessages = append(logMessages, log.Message)
						}

						Expect(logMessages).NotTo(ContainElement(ContainSubstring("ABQ")))
					})
				})
			})
		})
	})

	Context("with an erroring command", func() {
		var (
			exitCode                       int
			firstFailedTestDescription     string
			secondFailedTestDescription    string
			firstSuccessfulTestDescription string
			uploadedTestResults            []byte
		)

		BeforeEach(func() {
			exitCode = int(GinkgoRandomSeed() + 1)
			firstFailedTestDescription = fmt.Sprintf("first-failed-description-%d", GinkgoRandomSeed()+2)
			secondFailedTestDescription = fmt.Sprintf("second-failed-description-%d", GinkgoRandomSeed()+3)
			firstSuccessfulTestDescription = fmt.Sprintf("first-successful-description-%d", GinkgoRandomSeed()+4)

			mockGetExitStatusFromError := func(error) (int, error) {
				return exitCode, nil
			}
			service.TaskRunner.(*mocks.TaskRunner).MockGetExitStatusFromError = mockGetExitStatusFromError

			service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(r io.Reader) (
				*v1.TestResults,
				error,
			) {
				return &v1.TestResults{
					Tests: []v1.Test{
						{
							Name:     firstSuccessfulTestDescription,
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
								Status: v1.NewTimedOutTestStatus(),
							},
						},
					},
				}, nil
			}

			mockUploadTestResults := func(
				ctx context.Context,
				testSuite string,
				testResultsFiles []api.TestResultsFile,
			) ([]api.TestResultsUploadResult, error) {
				testResultsFileUploaded = true
				Expect(testResultsFiles).To(HaveLen(1))

				buf, err := io.ReadAll(testResultsFiles[0].FD)
				Expect(err).NotTo(HaveOccurred())
				uploadedTestResults = buf

				return []api.TestResultsUploadResult{
					{OriginalPaths: []string{testResultsFilePath}, Uploaded: true},
					{OriginalPaths: []string{"./fake/path/1.json", "./fake/path/2.json"}, Uploaded: false},
				}, nil
			}
			service.API.(*mocks.API).MockUploadTestResults = mockUploadTestResults
		})

		Context("no quarantined tests", func() {
			It("returns the error code of the command", func() {
				Expect(err).To(HaveOccurred())
				executionError, ok := errors.AsExecutionError(err)
				Expect(ok).To(BeTrue(), "Error is an execution error")
				Expect(executionError.Code).To(Equal(exitCode))
			})

			It("logs details about the file uploads", func() {
				logMessages := make([]string, 0)

				for _, log := range recordedLogs.All() {
					logMessages = append(logMessages, log.Message)
				}

				Expect(logMessages).To(ContainElement(ContainSubstring("Found 3 test result files:")))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- Uploaded %v", testResultsFilePath)))
				Expect(logMessages).To(ContainElement("- Unable to upload ./fake/path/1.json"))
				Expect(logMessages).To(ContainElement("- Unable to upload ./fake/path/2.json"))
			})

			Context("ABQ", func() {
				Context("with a supervisor ABQ state file", func() {
					BeforeEach(func() {
						abqStateFileExists = true
						abqStateJSON = fmt.Sprintf(`{
              "abq_version": "1.1.2",
              "abq_executable": "%s",
              "run_id": "not-a-real-run-id",
              "supervisor": true
            }`, abqExecutable)
					})

					It("calls abq set-exit-code", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeTrue())
						Expect(mockAbqCommandArgs).To(Equal([]string{
							"set-exit-code",
							"--run-id", "not-a-real-run-id",
							"--exit-code", "1",
						}))
						Expect(err).To(HaveOccurred())
					})
				})
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

			Context("ABQ", func() {
				Context("with a supervisor ABQ state file", func() {
					BeforeEach(func() {
						abqStateFileExists = true
						abqStateJSON = fmt.Sprintf(`{
              "abq_version": "1.1.2",
              "abq_executable": "%s",
              "run_id": "not-a-real-run-id",
              "supervisor": true
            }`, abqExecutable)
					})

					It("calls abq set-exit-code", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeTrue())
						Expect(mockAbqCommandArgs).To(Equal([]string{
							"set-exit-code",
							"--run-id", "not-a-real-run-id",
							"--exit-code", "1",
						}))
						Expect(err).To(HaveOccurred())
					})
				})
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

			Context("ABQ", func() {
				Context("with a supervisor ABQ state file", func() {
					BeforeEach(func() {
						abqStateFileExists = true
						abqStateJSON = fmt.Sprintf(`{
              "abq_version": "1.1.2",
              "abq_executable": "%s",
              "run_id": "not-a-real-run-id",
              "supervisor": true
            }`, abqExecutable)
					})

					It("calls abq set-exit-code", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeTrue())
						Expect(mockAbqCommandArgs).To(Equal([]string{
							"set-exit-code",
							"--run-id", "not-a-real-run-id",
							"--exit-code", "1",
						}))
						Expect(err).To(HaveOccurred())
					})
				})
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

			Context("ABQ", func() {
				Context("with a supervisor ABQ state file", func() {
					BeforeEach(func() {
						abqStateFileExists = true
						abqStateJSON = fmt.Sprintf(`{
              "abq_version": "1.1.2",
              "abq_executable": "%s",
              "run_id": "not-a-real-run-id",
              "supervisor": true
            }`, abqExecutable)
					})

					It("calls abq set-exit-code", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeTrue())
						Expect(mockAbqCommandArgs).To(Equal([]string{
							"set-exit-code",
							"--run-id", "not-a-real-run-id",
							"--exit-code", "1",
						}))
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Context("all tests quarantined tests fail", func() {
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

				testResults := &v1.TestResults{}
				err := json.Unmarshal(uploadedTestResults, testResults)
				Expect(err).To(BeNil())

				Expect(testResults.Summary.Quarantined).To(Equal(2))
				Expect(testResults.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))

				Expect(testResults.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(testResults.Tests[1].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusFailed))

				Expect(testResults.Tests[2].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(testResults.Tests[2].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusTimedOut))
			})

			Context("ABQ", func() {
				Context("with a supervisor ABQ state file", func() {
					BeforeEach(func() {
						abqStateFileExists = true
						abqStateJSON = fmt.Sprintf(`{
              "abq_version": "1.1.2",
              "abq_executable": "%s",
              "run_id": "not-a-real-run-id",
              "supervisor": true
            }`, abqExecutable)
					})

					It("calls abq set-exit-code", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeTrue())
						Expect(mockAbqCommandArgs).To(Equal([]string{
							"set-exit-code",
							"--run-id", "not-a-real-run-id",
							"--exit-code", "0",
						}))
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})
		})

		Context("all tests quarantined, but there are other errors", func() {
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

				service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(r io.Reader) (
					*v1.TestResults,
					error,
				) {
					return &v1.TestResults{
						OtherErrors: []v1.OtherError{
							{Message: "uh oh another error"},
						},
						Tests: []v1.Test{
							{
								Name:     firstSuccessfulTestDescription,
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

				Expect(logMessages).To(ContainElement(ContainSubstring("2 of 2 failures under quarantine")))
				Expect(logMessages).NotTo(ContainElement(ContainSubstring("remaining actionable")))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- %v", firstFailedTestDescription)))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- %v", secondFailedTestDescription)))

				testResults := &v1.TestResults{}
				err := json.Unmarshal(uploadedTestResults, testResults)
				Expect(err).To(BeNil())

				Expect(testResults.Summary.Quarantined).To(Equal(2))
				Expect(testResults.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))

				Expect(testResults.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(testResults.Tests[1].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusFailed))

				Expect(testResults.Tests[2].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(testResults.Tests[2].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusFailed))
			})

			Context("ABQ", func() {
				Context("with a supervisor ABQ state file", func() {
					BeforeEach(func() {
						abqStateFileExists = true
						abqStateJSON = fmt.Sprintf(`{
              "abq_version": "1.1.2",
              "abq_executable": "%s",
              "run_id": "not-a-real-run-id",
              "supervisor": true
            }`, abqExecutable)
					})

					It("calls abq set-exit-code", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeTrue())
						Expect(mockAbqCommandArgs).To(Equal([]string{
							"set-exit-code",
							"--run-id", "not-a-real-run-id",
							"--exit-code", "1",
						}))
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Context("some quarantined tests successful", func() {
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
						{
							CompositeIdentifier: fmt.Sprintf("%v -captain- %v", firstSuccessfulTestDescription, "/path/to/file.test"),
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

			It("reports the original status of success", func() {
				logMessages := make([]string, 0)

				for _, log := range recordedLogs.All() {
					logMessages = append(logMessages, log.Message)
				}

				Expect(logMessages).To(ContainElement(ContainSubstring("2 of 2 failures under quarantine")))
				Expect(logMessages).NotTo(ContainElement(ContainSubstring("remaining actionable")))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- %v", firstFailedTestDescription)))
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- %v", secondFailedTestDescription)))

				testResults := &v1.TestResults{}
				err := json.Unmarshal(uploadedTestResults, testResults)
				Expect(err).To(BeNil())

				Expect(testResults.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))

				Expect(testResults.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(testResults.Tests[1].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusFailed))

				Expect(testResults.Tests[2].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(testResults.Tests[2].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusTimedOut))
			})

			Context("ABQ", func() {
				Context("with a supervisor ABQ state file", func() {
					BeforeEach(func() {
						abqStateFileExists = true
						abqStateJSON = fmt.Sprintf(`{
              "abq_version": "1.1.2",
              "abq_executable": "%s",
              "run_id": "not-a-real-run-id",
              "supervisor": true
            }`, abqExecutable)
					})

					It("calls abq set-exit-code", func() {
						Expect(abqCommandStarted).To(BeTrue())
						Expect(abqCommandFinished).To(BeTrue())
						Expect(mockAbqCommandArgs).To(Equal([]string{
							"set-exit-code",
							"--run-id", "not-a-real-run-id",
							"--exit-code", "0",
						}))
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})
		})
	})
})
