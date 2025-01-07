package cli_test

import (
	"context"
	"fmt"
	"io"
	iofs "io/fs"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/backend/local"
	"github.com/rwx-research/captain-cli/internal/backend/remote"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/exec"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

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
		runConfig           cli.RunConfig

		testResultsFileUploaded, commandStarted, commandFinished, fetchedRunConfiguration bool
		abqStateFileExists, abqCommandStarted, abqCommandFinished                         bool
	)

	BeforeEach(func() {
		ctx = context.Background()
		err = nil
		testResultsFileUploaded = false
		commandStarted = false
		commandFinished = false
		fetchedRunConfiguration = false
		filesOpened = make([]string, 0)
		mockAbqCommandArgs = make([]string, 0)
		abqExecutable = "/path/to/abq"
		abqStateJSON = ""
		abqStateFileExists = false
		abqCommandStarted = false
		abqCommandFinished = false

		core, recordedLogs = observer.New(zapcore.InfoLevel)
		log := zaptest.NewLogger(GinkgoT(), zaptest.WrapOptions(
			zap.WrapCore(func(_ zapcore.Core) zapcore.Core { return core }),
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

		newCommand := func(_ context.Context, cfg exec.CommandConfig) (exec.Command, error) {
			switch cfg.Name {
			case abqExecutable:
				mockAbqCommandArgs = cfg.Args
				Expect(cfg.Env).To(HaveLen(0))
				return mockAbqCommand, nil
			case "retry":
				mockBundle := new(mocks.Command)
				mockBundle.MockStart = func() error {
					return nil
				}
				return mockBundle, nil
			default:
				Expect(cfg.Args).To(HaveLen(0))
				Expect(cfg.Name).To(Equal(arg))
				Expect(cfg.Env).To(HaveLen(2))
				Expect(cfg.Env).To(ContainElement("ABQ_SET_EXIT_CODE=false"))
				Expect(cfg.Env).To(ContainElement(ContainSubstring("ABQ_STATE_FILE=tmp/captain-abq-")))
				return mockCommand, nil
			}
		}
		service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand

		service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(_ io.Reader) (
			*v1.TestResults,
			error,
		) {
			return &v1.TestResults{}, nil
		}

		// Caution: This needs to be an existing file. We don't actually read from it, however the glob expansion
		// functionality is not mocked, i.e. it still uses the file-system.
		testResultsFilePath = "run.go"

		mockGlob := func(_ string) ([]string, error) {
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
		runConfig = cli.RunConfig{
			Command:             arg,
			TestResultsFileGlob: testResultsFilePath,
			SuiteID:             "test",
		}
	})

	JustBeforeEach(func() {
		err = service.RunSuite(ctx, runConfig)
	})

	Context("with an invalid run config", func() {
		BeforeEach(func() {
			runConfig = cli.RunConfig{
				Args:                 []string{arg},
				TestResultsFileGlob:  testResultsFilePath,
				SuiteID:              "test",
				Retries:              1,
				RetryCommandTemplate: "",
			}
		})

		It("errs", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing retry command"))
		})
	})

	Context("under expected conditions", func() {
		BeforeEach(func() {
			mockUploadTestResults := func(
				_ context.Context,
				_ string,
				_ v1.TestResults,
				_ bool,
			) ([]backend.TestResultsUploadResult, error) {
				testResultsFileUploaded = true
				return []backend.TestResultsUploadResult{{OriginalPaths: []string{testResultsFilePath}, Uploaded: true}}, nil
			}
			service.API.(*mocks.API).MockUpdateTestResults = mockUploadTestResults

			mockGetRunConfiguration := func(
				_ context.Context,
				_ string,
			) (backend.RunConfiguration, error) {
				fetchedRunConfiguration = true
				return backend.RunConfiguration{}, nil
			}
			service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration

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

		It("fetches the run configuration with the quarantined tests", func() {
			Expect(fetchedRunConfiguration).To(BeTrue())
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
			Expect(logMessages).To(ContainElement(ContainSubstring(
				fmt.Sprintf("- Updated Captain with results from %v", testResultsFilePath),
			)))
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

		Context("With updating disabled and a local client", func() {
			BeforeEach(func() {
				runConfig = cli.RunConfig{
					Command:             arg,
					TestResultsFileGlob: testResultsFilePath,
					SuiteID:             "test",
					UpdateStoredResults: false,
				}
				service.API = local.Client{}
			})

			It("runs the supplied command", func() {
				Expect(commandStarted).To(BeTrue())
			})

			It("waits until the supplied command stopped running", func() {
				Expect(commandFinished).To(BeTrue())
			})

			It("reads the test results file", func() {
				Expect(filesOpened).To(ContainElement(testResultsFilePath))
			})

			It("does not upload the test results file", func() {
				Expect(testResultsFileUploaded).To(BeFalse())
			})

			It("does not log the uploaded files", func() {
				logMessages := make([]string, 0)

				for _, log := range recordedLogs.All() {
					logMessages = append(logMessages, log.Message)
				}

				Expect(logMessages).NotTo(ContainElement(ContainSubstring("Found 1 test result file")))
				Expect(logMessages).NotTo(ContainElement(ContainSubstring(
					fmt.Sprintf("- Updated Captain with results from %v", testResultsFilePath),
				)))
			})
		})

		Context("With uploading disabled and a remote client", func() {
			var mockRoundTripper func(*http.Request) (*http.Response, error)

			BeforeEach(func() {
				runConfig = cli.RunConfig{
					Command:             arg,
					TestResultsFileGlob: testResultsFilePath,
					SuiteID:             "test",
					UploadResults:       false,
				}
				mockRoundTripper = func(req *http.Request) (*http.Response, error) {
					var resp http.Response

					Expect(req.Method).To(Equal(http.MethodGet))
					Expect(req.URL.Path).To(HaveSuffix("run_configuration"))

					resp.Body = io.NopCloser(strings.NewReader(`{}`))

					return &resp, nil
				}
				service.API = remote.Client{RoundTrip: mockRoundTripper}
			})

			It("runs the supplied command", func() {
				Expect(commandStarted).To(BeTrue())
			})

			It("waits until the supplied command stopped running", func() {
				Expect(commandFinished).To(BeTrue())
			})

			It("reads the test results file", func() {
				Expect(filesOpened).To(ContainElement(testResultsFilePath))
			})

			It("does not upload the test results file", func() {
				Expect(testResultsFileUploaded).To(BeFalse())
			})

			It("does not log the uploaded files", func() {
				logMessages := make([]string, 0)

				for _, log := range recordedLogs.All() {
					logMessages = append(logMessages, log.Message)
				}

				Expect(logMessages).NotTo(ContainElement(ContainSubstring("Found 1 test result file")))
				Expect(logMessages).NotTo(ContainElement(ContainSubstring(
					fmt.Sprintf("- Updated Captain with results from %v", testResultsFilePath),
				)))
			})
		})
	})

	Context("with an erroring command", func() {
		var (
			exitCode                       int
			firstFailedTestDescription     string
			secondFailedTestDescription    string
			firstSuccessfulTestDescription string
			uploadedTestResults            *v1.TestResults
		)

		BeforeEach(func() {
			uploadedTestResults = nil
			exitCode = int(GinkgoRandomSeed() + 1)
			firstFailedTestDescription = fmt.Sprintf("first-failed-description-%d", GinkgoRandomSeed()+2)
			secondFailedTestDescription = fmt.Sprintf("second-failed-description-%d", GinkgoRandomSeed()+3)
			firstSuccessfulTestDescription = fmt.Sprintf("first-successful-description-%d", GinkgoRandomSeed()+4)

			mockGetExitStatusFromError := func(error) (int, error) {
				return exitCode, nil
			}
			service.TaskRunner.(*mocks.TaskRunner).MockGetExitStatusFromError = mockGetExitStatusFromError

			service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(_ io.Reader) (
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
				_ context.Context,
				_ string,
				testResults v1.TestResults,
				_ bool,
			) ([]backend.TestResultsUploadResult, error) {
				testResultsFileUploaded = true

				uploadedTestResults = &testResults

				return []backend.TestResultsUploadResult{
					{OriginalPaths: []string{testResultsFilePath}, Uploaded: true},
					{OriginalPaths: []string{"./fake/path/1.json", "./fake/path/2.json"}, Uploaded: false},
				}, nil
			}
			service.API.(*mocks.API).MockUpdateTestResults = mockUploadTestResults
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
				Expect(logMessages).To(ContainElement(fmt.Sprintf("- Updated Captain with results from %v", testResultsFilePath)))
				Expect(logMessages).To(ContainElement("- Unable to update Captain with results from ./fake/path/1.json"))
				Expect(logMessages).To(ContainElement("- Unable to update Captain with results from ./fake/path/2.json"))
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
				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						QuarantinedTests: []backend.QuarantinedTest{
							{
								Test: backend.Test{
									CompositeIdentifier: fmt.Sprintf("%v -captain- %v", secondFailedTestDescription, "not/the/right/path.test"),
									IdentityComponents:  []string{"description", "file"},
									StrictIdentity:      true,
								},
							},
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
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
				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						QuarantinedTests: []backend.QuarantinedTest{
							{
								Test: backend.Test{
									CompositeIdentifier: "some-description -captain- huh",
									IdentityComponents:  []string{"description", "huh"},
									StrictIdentity:      true,
								},
							},
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
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
				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						QuarantinedTests: []backend.QuarantinedTest{
							{
								Test: backend.Test{
									CompositeIdentifier: fmt.Sprintf("%v -captain- %v", secondFailedTestDescription, "/other/path/to/file.test"),
									IdentityComponents:  []string{"description", "file"},
									StrictIdentity:      true,
								},
							},
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
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
				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						QuarantinedTests: []backend.QuarantinedTest{
							{
								Test: backend.Test{
									CompositeIdentifier: fmt.Sprintf("%v -captain- %v", firstFailedTestDescription, "/path/to/file.test"),
									IdentityComponents:  []string{"description", "file"},
									StrictIdentity:      true,
								},
							},
							{
								Test: backend.Test{
									CompositeIdentifier: fmt.Sprintf("%v -captain- %v", secondFailedTestDescription, "/other/path/to/file.test"),
									IdentityComponents:  []string{"description", "file"},
									StrictIdentity:      true,
								},
							},
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
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

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Quarantined).To(Equal(2))
				Expect(uploadedTestResults.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))

				Expect(uploadedTestResults.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(uploadedTestResults.Tests[1].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusFailed))

				Expect(uploadedTestResults.Tests[2].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(uploadedTestResults.Tests[2].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusTimedOut))
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

		Context("some quarantined tests successful", func() {
			BeforeEach(func() {
				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						QuarantinedTests: []backend.QuarantinedTest{
							{
								Test: backend.Test{
									CompositeIdentifier: fmt.Sprintf("%v -captain- %v", firstFailedTestDescription, "/path/to/file.test"),
									IdentityComponents:  []string{"description", "file"},
									StrictIdentity:      true,
								},
							},
							{
								Test: backend.Test{
									CompositeIdentifier: fmt.Sprintf("%v -captain- %v", secondFailedTestDescription, "/other/path/to/file.test"),
									IdentityComponents:  []string{"description", "file"},
									StrictIdentity:      true,
								},
							},
							{
								Test: backend.Test{
									CompositeIdentifier: fmt.Sprintf("%v -captain- %v", firstSuccessfulTestDescription, "/path/to/file.test"),
									IdentityComponents:  []string{"description", "file"},
									StrictIdentity:      true,
								},
							},
						},
					}, nil
				}
				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
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

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))

				Expect(uploadedTestResults.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(uploadedTestResults.Tests[1].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusFailed))

				Expect(uploadedTestResults.Tests[2].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(uploadedTestResults.Tests[2].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusTimedOut))
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

	Context("with other errors", func() {
		var (
			exitCode                       int
			firstFailedTestDescription     string
			secondFailedTestDescription    string
			firstSuccessfulTestDescription string
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

			service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(_ io.Reader) (
				*v1.TestResults,
				error,
			) {
				return &v1.TestResults{
					OtherErrors: []v1.OtherError{
						{
							Message: "Something went wrong",
						},
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
								Status: v1.NewTimedOutTestStatus(),
							},
						},
					},
				}, nil
			}

			mockUploadTestResults := func(
				_ context.Context,
				_ string,
				_ v1.TestResults,
				_ bool,
			) ([]backend.TestResultsUploadResult, error) {
				testResultsFileUploaded = true

				return []backend.TestResultsUploadResult{
					{OriginalPaths: []string{testResultsFilePath}, Uploaded: true},
					{OriginalPaths: []string{"./fake/path/1.json", "./fake/path/2.json"}, Uploaded: false},
				}, nil
			}
			service.API.(*mocks.API).MockUpdateTestResults = mockUploadTestResults

			mockGetRunConfiguration := func(
				_ context.Context,
				_ string,
			) (backend.RunConfiguration, error) {
				return backend.RunConfiguration{
					QuarantinedTests: []backend.QuarantinedTest{
						{
							Test: backend.Test{
								CompositeIdentifier: fmt.Sprintf("%v -captain- %v", firstFailedTestDescription, "/path/to/file.test"),
								IdentityComponents:  []string{"description", "file"},
								StrictIdentity:      true,
							},
						},
						{
							Test: backend.Test{
								CompositeIdentifier: fmt.Sprintf("%v -captain- %v", secondFailedTestDescription, "/other/path/to/file.test"),
								IdentityComponents:  []string{"description", "file"},
								StrictIdentity:      true,
							},
						},
					},
				}, nil
			}
			service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
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
		})

		It("logs the other error count", func() {
			logMessages := make([]string, 0)

			for _, log := range recordedLogs.All() {
				logMessages = append(logMessages, log.Message)
			}

			Expect(logMessages).To(ContainElement(ContainSubstring("1 other failure occurred")))
		})
	})

	Describe("retries", func() {
		var (
			parseCount            int
			exitCode              int
			uploadedTestResults   *v1.TestResults
			firstTestDescription  string
			secondTestDescription string
			thirdTestDescription  string
			firstInitialStatus    v1.TestStatus
			secondInitialStatus   v1.TestStatus
			thirdInitialStatus    v1.TestStatus
		)

		BeforeEach(func() {
			parseCount = 0
			exitCode = int(GinkgoRandomSeed() + 1)
			firstTestDescription = fmt.Sprintf("first-description-%d", GinkgoRandomSeed()+2)
			secondTestDescription = fmt.Sprintf("second-description-%d", GinkgoRandomSeed()+3)
			thirdTestDescription = fmt.Sprintf("third-description-%d", GinkgoRandomSeed()+4)
			firstInitialStatus = v1.NewFailedTestStatus(nil, nil, nil)
			secondInitialStatus = v1.NewFailedTestStatus(nil, nil, nil)
			thirdInitialStatus = v1.NewFailedTestStatus(nil, nil, nil)

			mockGetwd := func() (string, error) {
				return "/go/github.com/rwx-research/captain-cli", nil
			}
			service.FileSystem.(*mocks.FileSystem).MockGetwd = mockGetwd

			mockMkdirAll := func(_ string, _ os.FileMode) error {
				return nil
			}
			service.FileSystem.(*mocks.FileSystem).MockMkdirAll = mockMkdirAll

			mockRename := func(_, _ string) error {
				return nil
			}
			service.FileSystem.(*mocks.FileSystem).MockRename = mockRename

			mockMkdirTemp := func(_, _ string) (string, error) {
				return "/tmp/captain-test", nil
			}
			service.FileSystem.(*mocks.FileSystem).MockMkdirTemp = mockMkdirTemp

			mockStat := func(string) (iofs.FileInfo, error) {
				// abq state file
				return nil, iofs.ErrNotExist
			}
			service.FileSystem.(*mocks.FileSystem).MockStat = mockStat

			mockRemove := func(string) error {
				return nil
			}
			service.FileSystem.(*mocks.FileSystem).MockRemove = mockRemove

			mockGetExitStatusFromError := func(error) (int, error) {
				return exitCode, nil
			}
			service.TaskRunner.(*mocks.TaskRunner).MockGetExitStatusFromError = mockGetExitStatusFromError

			mockUploadTestResults := func(
				_ context.Context,
				_ string,
				testResults v1.TestResults,
				_ bool,
			) ([]backend.TestResultsUploadResult, error) {
				testResultsFileUploaded = true

				uploadedTestResults = &testResults

				return []backend.TestResultsUploadResult{
					{OriginalPaths: []string{testResultsFilePath}, Uploaded: true},
					{OriginalPaths: []string{"./fake/path/1.json", "./fake/path/2.json"}, Uploaded: false},
				}, nil
			}
			service.API.(*mocks.API).MockUpdateTestResults = mockUploadTestResults

			service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(_ io.Reader) (
				*v1.TestResults,
				error,
			) {
				parseCount++
				firstStatus := firstInitialStatus
				secondStatus := secondInitialStatus
				thirdStatus := thirdInitialStatus

				switch parseCount {
				case 2:
					secondStatus = v1.NewSuccessfulTestStatus()
					thirdStatus = v1.NewSuccessfulTestStatus()
				case 3:
					firstStatus = v1.NewSuccessfulTestStatus()
					secondStatus = v1.NewSuccessfulTestStatus()
					thirdStatus = v1.NewSuccessfulTestStatus()
				}

				return &v1.TestResults{
					Framework: v1.RubyRSpecFramework,
					Tests: []v1.Test{
						{
							ID:       &firstTestDescription,
							Name:     firstTestDescription,
							Location: &v1.Location{File: "/path/to/file.test"},
							Attempt: v1.TestAttempt{
								Status: firstStatus,
							},
						},
						{
							ID:       &secondTestDescription,
							Name:     secondTestDescription,
							Location: &v1.Location{File: "/path/to/file.test"},
							Attempt: v1.TestAttempt{
								Status: secondStatus,
							},
						},
						{
							ID:       &thirdTestDescription,
							Name:     thirdTestDescription,
							Location: &v1.Location{File: "/other/path/to/file.test"},
							Attempt: v1.TestAttempt{
								Status: thirdStatus,
							},
						},
					},
				}, nil
			}

			runConfig = cli.RunConfig{
				Command:              arg,
				TestResultsFileGlob:  testResultsFilePath,
				SuiteID:              "test",
				Retries:              1,
				RetryCommandTemplate: "retry {{ tests }}",
				SubstitutionsByFramework: map[v1.Framework]targetedretries.Substitution{
					v1.RubyRSpecFramework: new(targetedretries.RubyRSpecSubstitution),
				},
			}
		})

		Context("when there are no remaining failures after some retries", func() {
			BeforeEach(func() {
				runConfig.Retries = 5
			})

			It("stops retrying", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(0))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(3))

				Expect(uploadedTestResults.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
				Expect(uploadedTestResults.Tests[0].PastAttempts).To(HaveLen(2))
				Expect(uploadedTestResults.Tests[0].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))
				Expect(uploadedTestResults.Tests[0].PastAttempts[1].Status.Kind).To(Equal(v1.TestStatusFailed))

				Expect(uploadedTestResults.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
				Expect(uploadedTestResults.Tests[1].PastAttempts).To(HaveLen(2))
				Expect(uploadedTestResults.Tests[1].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))
				Expect(uploadedTestResults.Tests[1].PastAttempts[1].Status.Kind).To(Equal(v1.TestStatusSuccessful))

				Expect(uploadedTestResults.Tests[2].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
				Expect(uploadedTestResults.Tests[2].PastAttempts).To(HaveLen(2))
				Expect(uploadedTestResults.Tests[2].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))
				Expect(uploadedTestResults.Tests[2].PastAttempts[1].Status.Kind).To(Equal(v1.TestStatusSuccessful))
			})
		})

		Context("when there are pre- or post- retry commands", func() {
			var (
				preRetryCommandFinished, postRetryCommandFinished bool
				preRetryCommandStarted, postRetryCommandStarted   bool
			)

			BeforeEach(func() {
				runConfig.Retries = 1
				preRetryCommandStarted, preRetryCommandFinished = false, false
				postRetryCommandStarted, postRetryCommandFinished = false, false

				mockPreRetryCommand := new(mocks.Command)
				mockPreRetryCommand.MockStart = func() error {
					Expect(commandFinished).To(BeTrue(), "the original command should run before the first retry")

					commandStarted = false
					commandFinished = false
					preRetryCommandStarted = true

					return nil
				}
				mockPreRetryCommand.MockWait = func() error {
					Expect(preRetryCommandStarted).To(BeTrue())
					preRetryCommandFinished = true
					return nil
				}

				mockCommand.MockWait = func() error {
					commandFinished = true
					return nil
				}

				mockPostRetryCommand := new(mocks.Command)
				mockPostRetryCommand.MockStart = func() error {
					Expect(commandFinished).To(BeTrue())
					Expect(preRetryCommandFinished).To(BeTrue())
					postRetryCommandStarted = true
					return nil
				}
				mockPostRetryCommand.MockWait = func() error {
					Expect(postRetryCommandStarted).To(BeTrue())
					postRetryCommandFinished = true
					return nil
				}

				newCommand := func(_ context.Context, cfg exec.CommandConfig) (exec.Command, error) {
					switch cfg.Name {
					case "pre":
						Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_ATTEMPT_NUMBER=")))
						Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_INVOCATION_NUMBER=")))
						Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_COMMAND_ID=")))
						return mockPreRetryCommand, nil
					case "post":
						Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_ATTEMPT_NUMBER=")))
						Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_INVOCATION_NUMBER=")))
						Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_COMMAND_ID=")))
						return mockPostRetryCommand, nil
					default:
						if strings.Contains(cfg.Name, "retry") {
							Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_ATTEMPT_NUMBER=")))
							Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_INVOCATION_NUMBER=")))
							Expect(cfg.Env).To(ContainElement(ContainSubstring("CAPTAIN_RETRY_COMMAND_ID=")))
						} else {
							Expect(cfg.Env).NotTo(ContainElement(ContainSubstring("CAPTAIN_RETRY_ATTEMPT_NUMBER=")))
							Expect(cfg.Env).NotTo(ContainElement(ContainSubstring("CAPTAIN_RETRY_INVOCATION_NUMBER=")))
							Expect(cfg.Env).NotTo(ContainElement(ContainSubstring("CAPTAIN_RETRY_COMMAND_ID=")))
						}
						return mockCommand, nil
					}
				}
				service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand

				runConfig.PreRetryCommands = []string{"pre"}
				runConfig.PostRetryCommands = []string{"post"}
			})

			It("executes the pre-retry command before the command and the post-retry command afterwards", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(preRetryCommandStarted).To(BeTrue())
				Expect(preRetryCommandFinished).To(BeTrue())
				Expect(commandStarted).To(BeTrue())
				Expect(commandFinished).To(BeTrue())
				Expect(postRetryCommandStarted).To(BeTrue())
				Expect(postRetryCommandFinished).To(BeTrue())
			})
		})

		Context("when a intermediate artifacts path is defined", func() {
			var (
				intermediateTestResults []string
				mkdirCalled             bool
			)

			BeforeEach(func() {
				runConfig.IntermediateArtifactsPath = fmt.Sprintf("intermediate-results-%d", GinkgoRandomSeed())
				runConfig.Retries = 2

				mkdirCalled = false

				mockMkdirAll := func(dir string, _ os.FileMode) error {
					Expect(dir).To(ContainSubstring(runConfig.IntermediateArtifactsPath))
					mkdirCalled = true
					return nil
				}
				service.FileSystem.(*mocks.FileSystem).MockMkdirAll = mockMkdirAll

				mockRename := func(old, new string) error {
					Expect(old).To(Equal(testResultsFilePath))
					Expect(new).To(ContainSubstring(runConfig.IntermediateArtifactsPath))
					intermediateTestResults = append(intermediateTestResults, new)
					return nil
				}
				service.FileSystem.(*mocks.FileSystem).MockRename = mockRename

				mockStat := func(name string) (iofs.FileInfo, error) {
					switch name {
					case runConfig.IntermediateArtifactsPath:
						return mocks.FileInfo{Dir: true}, nil
					default:
						return nil, iofs.ErrNotExist
					}
				}
				service.FileSystem.(*mocks.FileSystem).MockStat = mockStat

				intermediateTestResults = make([]string, 0)
			})

			It("moves artifacts to the configured directory", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(mkdirCalled).To(BeTrue())
				Expect(intermediateTestResults).To(ContainElement(
					fmt.Sprintf("%s/original-attempt/%s", runConfig.IntermediateArtifactsPath, testResultsFilePath)),
				)
				Expect(intermediateTestResults).To(ContainElement(
					fmt.Sprintf("%s/retry-1/command-1/%s", runConfig.IntermediateArtifactsPath, testResultsFilePath)),
				)
				Expect(intermediateTestResults).To(ContainElement(
					fmt.Sprintf("%s/retry-2/command-1/%s", runConfig.IntermediateArtifactsPath, testResultsFilePath)),
				)
			})
		})

		Context("when there are failures left after all retries", func() {
			BeforeEach(func() {
				runConfig.Retries = 1
			})

			It("stops retrying", func() {
				Expect(err).To(HaveOccurred())
				executionError, ok := errors.AsExecutionError(err)
				Expect(ok).To(BeTrue(), "Error is an execution error")
				Expect(executionError.Code).To(Equal(exitCode))

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(2))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(1))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(3))

				Expect(uploadedTestResults.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusFailed))
				Expect(uploadedTestResults.Tests[0].PastAttempts).To(HaveLen(1))
				Expect(uploadedTestResults.Tests[0].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))

				Expect(uploadedTestResults.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
				Expect(uploadedTestResults.Tests[1].PastAttempts).To(HaveLen(1))
				Expect(uploadedTestResults.Tests[1].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))

				Expect(uploadedTestResults.Tests[2].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
				Expect(uploadedTestResults.Tests[2].PastAttempts).To(HaveLen(1))
				Expect(uploadedTestResults.Tests[2].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))
			})
		})

		Context("when quarantining is set up", func() {
			BeforeEach(func() {
				runConfig.Retries = 1

				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						QuarantinedTests: []backend.QuarantinedTest{
							{
								Test: backend.Test{
									CompositeIdentifier: fmt.Sprintf("%v -captain- %v", firstTestDescription, "/path/to/file.test"),
									IdentityComponents:  []string{"description", "file"},
									StrictIdentity:      true,
								},
							},
						},
					}, nil
				}

				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
			})

			It("quarantines any remaining failures that are marked as such", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(2))
				Expect(uploadedTestResults.Summary.Quarantined).To(Equal(1))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(0))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(3))

				Expect(uploadedTestResults.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusQuarantined))
				Expect(uploadedTestResults.Tests[0].Attempt.Status.OriginalStatus.Kind).To(Equal(v1.TestStatusFailed))
				Expect(uploadedTestResults.Tests[0].PastAttempts).To(HaveLen(1))
				Expect(uploadedTestResults.Tests[0].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))

				Expect(uploadedTestResults.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
				Expect(uploadedTestResults.Tests[1].PastAttempts).To(HaveLen(1))
				Expect(uploadedTestResults.Tests[1].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))

				Expect(uploadedTestResults.Tests[2].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
				Expect(uploadedTestResults.Tests[2].PastAttempts).To(HaveLen(1))
				Expect(uploadedTestResults.Tests[2].PastAttempts[0].Status.Kind).To(Equal(v1.TestStatusFailed))
			})
		})

		Context("when retrying only flaky tests and there are no flaky tests", func() {
			BeforeEach(func() {
				runConfig.Retries = -1
				runConfig.FlakyRetries = 1

				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{FlakyTests: []backend.Test{}}, nil
				}

				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
			})

			It("does not retry", func() {
				Expect(err).To(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(3))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when retrying only flaky tests and only a subset of the failing tests are flaky", func() {
			BeforeEach(func() {
				runConfig.Retries = -1
				runConfig.FlakyRetries = 1

				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						FlakyTests: []backend.Test{
							{
								CompositeIdentifier: firstTestDescription,
								IdentityComponents:  []string{"description"},
								StrictIdentity:      true,
							},
						},
					}, nil
				}

				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration

				newCommand := func(_ context.Context, cfg exec.CommandConfig) (exec.Command, error) {
					if cfg.Name == "retry" {
						Expect(cfg.Args).To(ContainElement(ContainSubstring(firstTestDescription)))
						Expect(cfg.Args).NotTo(ContainElement(ContainSubstring(secondTestDescription)))
						Expect(cfg.Args).NotTo(ContainElement(ContainSubstring(thirdTestDescription)))
					}

					return mockCommand, nil
				}
				service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand
			})

			It("only retries the flaky ones", func() {
				// see assertions in newCommand
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when retrying flaky more than the non-flaky tests", func() {
			var retryCount int

			BeforeEach(func() {
				runConfig.Retries = 1
				runConfig.FlakyRetries = 2
				retryCount = 0

				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						FlakyTests: []backend.Test{
							{
								CompositeIdentifier: firstTestDescription,
								IdentityComponents:  []string{"description"},
								StrictIdentity:      true,
							},
						},
					}, nil
				}

				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration

				newCommand := func(_ context.Context, cfg exec.CommandConfig) (exec.Command, error) {
					if cfg.Name == "retry" {
						retryCount++

						if retryCount == 1 {
							Expect(cfg.Args).To(ContainElement(ContainSubstring(firstTestDescription)))
							Expect(cfg.Args).To(ContainElement(ContainSubstring(secondTestDescription)))
							Expect(cfg.Args).To(ContainElement(ContainSubstring(thirdTestDescription)))
						} else {
							Expect(cfg.Args).To(ContainElement(ContainSubstring(firstTestDescription)))
							Expect(cfg.Args).NotTo(ContainElement(ContainSubstring(secondTestDescription)))
							Expect(cfg.Args).NotTo(ContainElement(ContainSubstring(thirdTestDescription)))
						}
					}

					return mockCommand, nil
				}
				service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand

				service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(_ io.Reader) (
					*v1.TestResults,
					error,
				) {
					parseCount++

					firstStatus := firstInitialStatus
					secondStatus := secondInitialStatus
					thirdStatus := thirdInitialStatus

					return &v1.TestResults{
						Framework: v1.RubyRSpecFramework,
						Tests: []v1.Test{
							{
								ID:       &firstTestDescription,
								Name:     firstTestDescription,
								Location: &v1.Location{File: "/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: firstStatus,
								},
							},
							{
								ID:       &secondTestDescription,
								Name:     secondTestDescription,
								Location: &v1.Location{File: "/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: secondStatus,
								},
							},
							{
								ID:       &thirdTestDescription,
								Name:     thirdTestDescription,
								Location: &v1.Location{File: "/other/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: thirdStatus,
								},
							},
						},
					}, nil
				}
			})

			It("stops retrying the non-flaky ones after it exhausts the attempts", func() {
				// see assertions in newCommand
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when retrying non-flaky more than the flaky tests", func() {
			var retryCount int

			BeforeEach(func() {
				runConfig.Retries = 2
				runConfig.FlakyRetries = 1
				retryCount = 0

				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						FlakyTests: []backend.Test{
							{
								CompositeIdentifier: firstTestDescription,
								IdentityComponents:  []string{"description"},
								StrictIdentity:      true,
							},
						},
					}, nil
				}

				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration

				newCommand := func(_ context.Context, cfg exec.CommandConfig) (exec.Command, error) {
					if cfg.Name == "retry" {
						retryCount++

						if retryCount == 1 {
							Expect(cfg.Args).To(ContainElement(ContainSubstring(firstTestDescription)))
							Expect(cfg.Args).To(ContainElement(ContainSubstring(secondTestDescription)))
							Expect(cfg.Args).To(ContainElement(ContainSubstring(thirdTestDescription)))
						} else {
							Expect(cfg.Args).NotTo(ContainElement(ContainSubstring(firstTestDescription)))
							Expect(cfg.Args).To(ContainElement(ContainSubstring(secondTestDescription)))
							Expect(cfg.Args).To(ContainElement(ContainSubstring(thirdTestDescription)))
						}
					}

					return mockCommand, nil
				}
				service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand

				service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(_ io.Reader) (
					*v1.TestResults,
					error,
				) {
					parseCount++

					firstStatus := firstInitialStatus
					secondStatus := secondInitialStatus
					thirdStatus := thirdInitialStatus

					return &v1.TestResults{
						Framework: v1.RubyRSpecFramework,
						Tests: []v1.Test{
							{
								ID:       &firstTestDescription,
								Name:     firstTestDescription,
								Location: &v1.Location{File: "/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: firstStatus,
								},
							},
							{
								ID:       &secondTestDescription,
								Name:     secondTestDescription,
								Location: &v1.Location{File: "/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: secondStatus,
								},
							},
							{
								ID:       &thirdTestDescription,
								Name:     thirdTestDescription,
								Location: &v1.Location{File: "/other/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: thirdStatus,
								},
							},
						},
					}, nil
				}
			})

			It("stops retrying the flaky ones after it exhausts the attempts", func() {
				// see assertions in newCommand
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when retrying everything and there are flaky tests", func() {
			BeforeEach(func() {
				runConfig.Retries = 2
				runConfig.FlakyRetries = -1

				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						FlakyTests: []backend.Test{
							{
								CompositeIdentifier: firstTestDescription,
								IdentityComponents:  []string{"description"},
								StrictIdentity:      true,
							},
						},
					}, nil
				}

				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration

				newCommand := func(_ context.Context, cfg exec.CommandConfig) (exec.Command, error) {
					if cfg.Name == "retry" {
						Expect(cfg.Args).To(ContainElement(ContainSubstring(firstTestDescription)))
						Expect(cfg.Args).To(ContainElement(ContainSubstring(secondTestDescription)))
						Expect(cfg.Args).To(ContainElement(ContainSubstring(thirdTestDescription)))
					}

					return mockCommand, nil
				}
				service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand

				service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(_ io.Reader) (
					*v1.TestResults,
					error,
				) {
					parseCount++

					firstStatus := firstInitialStatus
					secondStatus := secondInitialStatus
					thirdStatus := thirdInitialStatus

					return &v1.TestResults{
						Framework: v1.RubyRSpecFramework,
						Tests: []v1.Test{
							{
								ID:       &firstTestDescription,
								Name:     firstTestDescription,
								Location: &v1.Location{File: "/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: firstStatus,
								},
							},
							{
								ID:       &secondTestDescription,
								Name:     secondTestDescription,
								Location: &v1.Location{File: "/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: secondStatus,
								},
							},
							{
								ID:       &thirdTestDescription,
								Name:     thirdTestDescription,
								Location: &v1.Location{File: "/other/path/to/file.test"},
								Attempt: v1.TestAttempt{
									Status: thirdStatus,
								},
							},
						},
					}, nil
				}
			})

			It("retries all tests until the attempts are exhausted, flaky or not", func() {
				// see assertions in newCommand
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when failing fast and there are non-flaky failures that can't pass", func() {
			BeforeEach(func() {
				runConfig.Retries = -1
				runConfig.FlakyRetries = 1
				runConfig.FailRetriesFast = true

				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						FlakyTests: []backend.Test{
							{
								CompositeIdentifier: firstTestDescription,
								IdentityComponents:  []string{"description"},
								StrictIdentity:      true,
							},
						},
					}, nil
				}

				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
			})

			It("fails quickly", func() {
				Expect(err).To(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(3))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when failing fast and there are non-flaky failures that pass fast enough", func() {
			BeforeEach(func() {
				runConfig.Retries = 1
				runConfig.FlakyRetries = 2
				runConfig.FailRetriesFast = true

				mockGetRunConfiguration := func(
					_ context.Context,
					_ string,
				) (backend.RunConfiguration, error) {
					return backend.RunConfiguration{
						FlakyTests: []backend.Test{
							{
								CompositeIdentifier: firstTestDescription,
								IdentityComponents:  []string{"description"},
								StrictIdentity:      true,
							},
						},
					}, nil
				}

				service.API.(*mocks.API).MockGetRunConfiguration = mockGetRunConfiguration
			})

			It("finishes retrying", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(0))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(3))
			})
		})

		Context("when there are too many failures by percent", func() {
			BeforeEach(func() {
				runConfig.MaxTestsToRetry = "50%"
				runConfig.Retries = 2
			})

			It("does not retry", func() {
				Expect(err).To(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(3))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when there are not too many failures by percent", func() {
			BeforeEach(func() {
				runConfig.MaxTestsToRetry = "200%"
				runConfig.Retries = 2
			})

			It("retries", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(0))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(3))
			})
		})

		Context("when there are too many failures by count", func() {
			BeforeEach(func() {
				runConfig.MaxTestsToRetry = "1"
				runConfig.Retries = 2
			})

			It("does not retry", func() {
				Expect(err).To(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(3))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when there are not too many failures by count", func() {
			BeforeEach(func() {
				runConfig.MaxTestsToRetry = "5"
				runConfig.Retries = 2
			})

			It("retries", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(0))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(3))
			})
		})

		Context("when using a custom command w/ JSON even though the framework is supported", func() {
			var removedFiles []string

			BeforeEach(func() {
				service.FileSystem.(*mocks.FileSystem).MockRemove = func(name string) error {
					removedFiles = append(removedFiles, name)
					return nil
				}

				service.FileSystem.(*mocks.FileSystem).MockCreateTemp = func(_ string, _ string) (fs.File, error) {
					mockFile := new(mocks.File)
					mockFile.Builder = new(strings.Builder)
					mockFile.MockName = func() string {
						return "json-temp-file"
					}
					return mockFile, nil
				}
				runConfig.Retries = 2
				runConfig.RetryCommandTemplate = "retry {{ jsonFilePath }}"

				newCommand := func(_ context.Context, cfg exec.CommandConfig) (exec.Command, error) {
					if cfg.Name == "retry" {
						Expect(cfg.Args).To(ContainElement(ContainSubstring("json-temp-file")))
					}

					return mockCommand, nil
				}
				service.TaskRunner.(*mocks.TaskRunner).MockNewCommand = newCommand
			})

			It("passes the JSON file to the retry command and cleans up", func() {
				// see assertions in newCommand
				Expect(err).NotTo(HaveOccurred())
				Expect(removedFiles).To(ContainElement("json-temp-file"))
			})
		})

		Context("when there are no failures", func() {
			BeforeEach(func() {
				firstInitialStatus = v1.NewSuccessfulTestStatus()
				secondInitialStatus = v1.NewSuccessfulTestStatus()
				thirdInitialStatus = v1.NewSuccessfulTestStatus()
			})

			It("does not retry", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(3))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(0))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when abq has run", func() {
			BeforeEach(func() {
				mockStat := func(name string) (iofs.FileInfo, error) {
					const abqStateSubstring string = "tmp/captain-abq-"

					Expect(name).To(ContainSubstring(abqStateSubstring))
					mockedFile := new(mocks.File)
					return mockedFile, nil
				}

				service.FileSystem.(*mocks.FileSystem).MockStat = mockStat
			})

			It("does not retry", func() {
				Expect(err).To(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(0))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(3))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when the template cannot compile", func() {
			BeforeEach(func() {
				runConfig.RetryCommandTemplate = "bundle exec rspec {{ tests }} {{ tests }}"
			})

			It("does not retry", func() {
				Expect(err).To(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(0))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(3))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when the framework does not have substitutions yet", func() {
			BeforeEach(func() {
				runConfig.SubstitutionsByFramework = map[v1.Framework]targetedretries.Substitution{}
			})

			It("does not retry", func() {
				Expect(err).To(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(0))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(3))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when the template is not valid for the framework", func() {
			BeforeEach(func() {
				runConfig.RetryCommandTemplate = "bundle exec rspec {{ wat }}"
			})

			It("does not retry", func() {
				Expect(err).To(HaveOccurred())

				Expect(uploadedTestResults).ToNot(BeNil())
				Expect(uploadedTestResults.Summary.Tests).To(Equal(3))
				Expect(uploadedTestResults.Summary.Successful).To(Equal(0))
				Expect(uploadedTestResults.Summary.Failed).To(Equal(3))
				Expect(uploadedTestResults.Summary.Retries).To(Equal(0))
			})
		})

		Context("when there are no results", func() {
			BeforeEach(func() {
				mockGlob := func(_ string) ([]string, error) {
					return []string{}, nil
				}
				service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
			})

			It("does not retry", func() {
				Expect(err).To(HaveOccurred())
				executionError, ok := errors.AsExecutionError(err)
				Expect(ok).To(BeTrue(), "Error is an execution error")
				Expect(executionError.Code).To(Equal(exitCode))
			})
		})
	})

	Context("when there are multiple frameworks parsed", func() {
		var parseCount int

		BeforeEach(func() {
			parseCount = 0

			mockGetExitStatusFromError := func(error) (int, error) {
				return 0, nil
			}
			service.TaskRunner.(*mocks.TaskRunner).MockGetExitStatusFromError = mockGetExitStatusFromError

			mockGlob := func(_ string) ([]string, error) {
				return []string{testResultsFilePath, testResultsFilePath}, nil
			}
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob

			service.ParseConfig.MutuallyExclusiveParsers[0].(*mocks.Parser).MockParse = func(_ io.Reader) (
				*v1.TestResults,
				error,
			) {
				parseCount++

				framework := v1.RubyRSpecFramework
				if parseCount == 2 {
					framework = v1.JavaScriptJestFramework
				}

				return &v1.TestResults{
					Framework: framework,
					Tests:     []v1.Test{},
				}, nil
			}
		})

		It("exits non-zero and logs an error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Multiple frameworks detected. The captain CLI only works with one framework at a time",
			))
		})
	})
})
