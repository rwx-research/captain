package parsing_test

import (
	"encoding/base64"
	"io"
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
	return &v1.TestResults{Framework: v1.NewOtherFramework(&one, &one)}, nil
}

type SuccessfulParserTwo struct{}

func (p SuccessfulParserTwo) Parse(testResults io.Reader) (*v1.TestResults, error) {
	buf, err := io.ReadAll(testResults)
	Expect(string(buf)).To(Equal("the fake contents to base64 encode"))
	Expect(err).NotTo(HaveOccurred())
	two := "two"
	return &v1.TestResults{Framework: v1.NewOtherFramework(&two, &two)}, nil
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

	It("is an error when no parsers are provided", func() {
		results, err := parsing.Parse(file, make([]parsing.Parser, 0), log)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("No parsers were provided"))
	})

	It("is an error when no logger is provided", func() {
		results, err := parsing.Parse(file, []parsing.Parser{SuccessfulParserOne{}}, nil)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("No logger was provided"))
	})

	It("is an error when a parser returns neither a result nor an error", func() {
		results, err := parsing.Parse(file, []parsing.Parser{NeitherErrorNorResultsParser{}}, log)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("NeitherErrorNorResultsParser did not error and did not return a test result"),
		)
	})

	It("is an error when no parsers can parse", func() {
		results, err := parsing.Parse(
			file,
			[]parsing.Parser{
				ErrorParser{},
				ErrorParser{},
				ErrorParser{},
			},
			log,
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
			[]parsing.Parser{
				SuccessfulParserTwo{},
				ErrorParser{},
				SuccessfulParserOne{},
			},
			log,
		)

		Expect(results).NotTo(BeNil())
		Expect(*results.Framework.ProvidedKind).To(Equal("two"))
		Expect(results.DerivedFrom).To(Equal(
			[]v1.OriginalTestResults{
				{
					OriginalFilePath: "some/path/to/file",
					Contents:         base64.StdEncoding.EncodeToString([]byte("the fake contents to base64 encode")),
					GroupNumber:      1,
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
	})
})
