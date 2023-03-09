package parsing_test

import (
	"encoding/base64"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type SuccessfulParserOne struct{}

func (p SuccessfulParserOne) Parse(testResults io.Reader) (*v1.TestResults, error) {
	buf, err := io.ReadAll(testResults)
	Expect(string(buf)).To(Equal("the fake contents to base64 encode"))
	Expect(err).NotTo(HaveOccurred())
	one := "one"
	return &v1.TestResults{Summary: v1.Summary{Tests: 1}, Framework: v1.NewOtherFramework(&one, &one)}, nil
}

type SuccessfulParserTwo struct{}

func (p SuccessfulParserTwo) Parse(testResults io.Reader) (*v1.TestResults, error) {
	buf, err := io.ReadAll(testResults)
	Expect(string(buf)).To(Equal("the fake contents to base64 encode"))
	Expect(err).NotTo(HaveOccurred())
	two := "two"
	return &v1.TestResults{Summary: v1.Summary{Tests: 2}, Framework: v1.NewOtherFramework(&two, &two)}, nil
}

type ErrorParser struct{}

func (p ErrorParser) Parse(testResults io.Reader) (*v1.TestResults, error) {
	buf, err := io.ReadAll(testResults)
	Expect(string(buf)).To(Equal("the fake contents to base64 encode"))
	Expect(err).NotTo(HaveOccurred())
	return nil, errors.NewInternalError("could not parse")
}

type NeitherErrorNorResultsParser struct{}

func (p NeitherErrorNorResultsParser) Parse(testResults io.Reader) (*v1.TestResults, error) {
	return nil, nil
}

var _ = Describe("Parse", func() {
	var (
		logCore      zapcore.Core
		log          *zap.SugaredLogger
		recordedLogs *observer.ObservedLogs
		file         *mocks.File
	)

	BeforeEach(func() {
		logCore, recordedLogs = observer.New(zapcore.DebugLevel)
		log = zaptest.NewLogger(GinkgoT(), zaptest.WrapOptions(
			zap.WrapCore(func(original zapcore.Core) zapcore.Core { return logCore }),
		)).Sugar()
		file = new(mocks.File)
		file.Reader = strings.NewReader("the fake contents to base64 encode")
		file.MockName = func() string { return "some/path/to/file" }
	})

	It("is an error when no logger is provided", func() {
		results, err := parsing.Parse(
			file,
			1,
			parsing.Config{},
		)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("No logger was provided"))
	})

	It("is an error when only language is provided", func() {
		results, err := parsing.Parse(
			file,
			1,
			parsing.Config{Logger: log, ProvidedFrameworkLanguage: "foo"},
		)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("when specifying a language, the framework also needs to be provided"))
	})

	It("is an error when only kind is provided", func() {
		results, err := parsing.Parse(
			file,
			1,
			parsing.Config{Logger: log, ProvidedFrameworkKind: "foo"},
		)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("when specifying a framework, the language also needs to be provided"))
	})

	It("is an error when a parser returns neither a result nor an error", func() {
		results, err := parsing.Parse(
			file,
			1,
			parsing.Config{
				MutuallyExclusiveParsers: []parsing.Parser{NeitherErrorNorResultsParser{}},
				Logger:                   log,
			},
		)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("NeitherErrorNorResultsParser did not error and did not return a test result"),
		)
	})

	It("is an error when no parsers can parse", func() {
		results, err := parsing.Parse(
			file,
			1,
			parsing.Config{
				MutuallyExclusiveParsers: []parsing.Parser{
					ErrorParser{},
					ErrorParser{},
					ErrorParser{},
				},
				Logger: log,
			},
		)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("No parsers were capable of parsing the provided test results"),
		)

		logMessages := make([]string, 0)
		for _, log := range recordedLogs.All() {
			logMessages = append(logMessages, log.Message)
		}

		Expect(logMessages).To(ContainElement(
			ContainSubstring("ErrorParser was not capable of parsing the test results"),
		))
		Expect(logMessages).NotTo(ContainElement(
			ContainSubstring("ultimately responsible for parsing the test results"),
		))
	})

	It("returns the first test results with the base64 encoded content", func() {
		results, err := parsing.Parse(
			file,
			2,
			parsing.Config{
				MutuallyExclusiveParsers: []parsing.Parser{
					SuccessfulParserTwo{},
					ErrorParser{},
					SuccessfulParserOne{},
				},
				Logger: log,
			},
		)

		Expect(results).NotTo(BeNil())
		Expect(*results.Framework.ProvidedKind).To(Equal("two"))
		Expect(results.DerivedFrom).To(Equal(
			[]v1.OriginalTestResults{
				{
					OriginalFilePath: "some/path/to/file",
					Contents:         base64.StdEncoding.EncodeToString([]byte("the fake contents to base64 encode")),
					GroupNumber:      2,
				},
			},
		))
		Expect(err).To(BeNil())

		logMessages := make([]string, 0)
		for _, log := range recordedLogs.All() {
			logMessages = append(logMessages, log.Message)
		}

		Expect(logMessages).To(ContainElement(
			ContainSubstring("ErrorParser was not capable of parsing the test results"),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("SuccessfulParserOne was capable of parsing the test results."),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("SuccessfulParserTwo was capable of parsing the test results."),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("SuccessfulParserTwo was ultimately responsible for parsing the test results"),
		))

		// ensure it rewinds the file once done
		buf, err := io.ReadAll(file)
		Expect(string(buf)).To(Equal("the fake contents to base64 encode"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("does not set DerivedFrom when parsing with the RWX parser", func() {
		fixture, err := os.Open("../../test/fixtures/rwx/v1_not_derived.json")
		Expect(err).ToNot(HaveOccurred())
		buf, err := io.ReadAll(fixture)
		Expect(err).ToNot(HaveOccurred())

		file = new(mocks.File)
		file.Reader = strings.NewReader(string(buf))
		file.MockName = func() string { return "some/path/to/file" }

		results, err := parsing.Parse(
			file,
			1,
			parsing.Config{
				MutuallyExclusiveParsers: []parsing.Parser{parsing.RWXParser{}},
				Logger:                   log,
			},
		)

		Expect(results).NotTo(BeNil())
		Expect(results.DerivedFrom).To(BeNil())
		Expect(err).To(BeNil())
	})

	Describe("when no language and kind are provided", func() {
		It("uses the mutually exclusive and generic parsers to auto-detect framework", func() {
			results, err := parsing.Parse(
				file,
				1,
				parsing.Config{
					ProvidedFrameworkLanguage: "",
					ProvidedFrameworkKind:     "",
					MutuallyExclusiveParsers: []parsing.Parser{
						SuccessfulParserTwo{},
						ErrorParser{},
					},
					GenericParsers: []parsing.Parser{SuccessfulParserOne{}},
					Logger:         log,
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeNil())
			Expect(*results.Framework.ProvidedKind).To(Equal("two"))

			results, err = parsing.Parse(
				file,
				1,
				parsing.Config{
					ProvidedFrameworkLanguage: "",
					ProvidedFrameworkKind:     "",
					MutuallyExclusiveParsers: []parsing.Parser{
						ErrorParser{},
					},
					GenericParsers: []parsing.Parser{SuccessfulParserOne{}},
					Logger:         log,
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeNil())
			Expect(*results.Framework.ProvidedKind).To(Equal("one"))
		})

		It("is an error when no parsers are provided", func() {
			results, err := parsing.Parse(
				file,
				1,
				parsing.Config{
					ProvidedFrameworkLanguage: "",
					ProvidedFrameworkKind:     "",
					Logger:                    log,
				},
			)

			Expect(results).To(BeNil())
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("No parsers were provided"))
		})
	})

	Describe("when an unknown language and kind are provided", func() {
		It("uses the generic parsers and sets the provided language and kind", func() {
			results, err := parsing.Parse(
				file,
				1,
				parsing.Config{
					ProvidedFrameworkLanguage: "bar",
					ProvidedFrameworkKind:     "foo",
					MutuallyExclusiveParsers: []parsing.Parser{
						SuccessfulParserTwo{},
						ErrorParser{},
					},
					GenericParsers: []parsing.Parser{SuccessfulParserOne{}},
					Logger:         log,
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeNil())
			Expect(results.Framework.IsOther()).To(Equal(true))
			Expect(*results.Framework.ProvidedLanguage).To(Equal("bar"))
			Expect(*results.Framework.ProvidedKind).To(Equal("foo"))
		})

		It("is an error when no parsers are provided", func() {
			results, err := parsing.Parse(
				file,
				1,
				parsing.Config{
					ProvidedFrameworkLanguage: "bar",
					ProvidedFrameworkKind:     "foo",
					Logger:                    log,
				},
			)

			Expect(results).To(BeNil())
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("No parsers were provided"))
		})
	})

	Describe("when a known language and kind are provided", func() {
		It("uses the framework parsers", func() {
			results, err := parsing.Parse(
				file,
				1,
				parsing.Config{
					ProvidedFrameworkLanguage: "RuBy",
					ProvidedFrameworkKind:     "RspEc",
					FrameworkParsers: map[v1.Framework][]parsing.Parser{
						v1.RubyRSpecFramework:         {SuccessfulParserOne{}},
						v1.JavaScriptCypressFramework: {SuccessfulParserTwo{}},
					},
					Logger: log,
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeNil())
			Expect(results.Summary.Tests).To(Equal(1))
			Expect(results.Framework).To(Equal(v1.RubyRSpecFramework))

			results, err = parsing.Parse(
				file,
				1,
				parsing.Config{
					ProvidedFrameworkLanguage: "javascript",
					ProvidedFrameworkKind:     "cypress",
					FrameworkParsers: map[v1.Framework][]parsing.Parser{
						v1.RubyRSpecFramework:         {SuccessfulParserOne{}},
						v1.JavaScriptCypressFramework: {SuccessfulParserTwo{}},
					},
					Logger: log,
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeNil())
			Expect(results.Summary.Tests).To(Equal(2))
			Expect(results.Framework).To(Equal(v1.JavaScriptCypressFramework))
		})

		It("is an error when no parsers are provided", func() {
			results, err := parsing.Parse(
				file,
				1,
				parsing.Config{
					ProvidedFrameworkLanguage: "RspEc",
					ProvidedFrameworkKind:     "RuBy",
					Logger:                    log,
				},
			)

			Expect(results).To(BeNil())
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("No parsers were provided"))
		})
	})
})
