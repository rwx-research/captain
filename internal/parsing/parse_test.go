package parsing_test

import (
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type PositiveSentimentParser struct{}

func (p PositiveSentimentParser) Parse(testResults io.Reader) (*parsing.ParseResult, error) {
	positive := "positive"
	return &parsing.ParseResult{
		Sentiment:   parsing.PositiveParseResultSentiment,
		TestResults: v1.TestResults{Framework: v1.NewOtherFramework(&positive, &positive)},
		Parser:      p,
	}, nil
}

type NeutralSentimentParser struct{}

func (p NeutralSentimentParser) Parse(testResults io.Reader) (*parsing.ParseResult, error) {
	neutral := "neutral"
	return &parsing.ParseResult{
		Sentiment:   parsing.NeutralParseResultSentiment,
		TestResults: v1.TestResults{Framework: v1.NewOtherFramework(&neutral, &neutral)},
		Parser:      p,
	}, nil
}

type NegativeSentimentParser struct{}

func (p NegativeSentimentParser) Parse(testResults io.Reader) (*parsing.ParseResult, error) {
	negative := "negative"
	return &parsing.ParseResult{
		Sentiment:   parsing.NegativeParseResultSentiment,
		TestResults: v1.TestResults{Framework: v1.NewOtherFramework(&negative, &negative)},
		Parser:      p,
	}, nil
}

type ErrorParser struct{}

func (p ErrorParser) Parse(testResults io.Reader) (*parsing.ParseResult, error) {
	return nil, errors.NewInternalError("could not parse")
}

type NeitherErrorNorResultsParser struct{}

func (p NeitherErrorNorResultsParser) Parse(testResults io.Reader) (*parsing.ParseResult, error) {
	return nil, nil
}

var _ = Describe("Parse", func() {
	var (
		logCore      zapcore.Core
		log          *zap.SugaredLogger
		recordedLogs *observer.ObservedLogs
		testResults  io.Reader
	)

	BeforeEach(func() {
		logCore, recordedLogs = observer.New(zapcore.DebugLevel)
		log = zaptest.NewLogger(GinkgoT(), zaptest.WrapOptions(
			zap.WrapCore(func(original zapcore.Core) zapcore.Core { return logCore }),
		)).Sugar()
		testResults = strings.NewReader("")
	})

	It("is an error when no parsers are provided", func() {
		results, err := parsing.Parse(testResults, make([]parsing.Parser, 0), log)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("No parsers were provided"))
	})

	It("is an error when no logger is provided", func() {
		results, err := parsing.Parse(testResults, []parsing.Parser{NeutralSentimentParser{}}, nil)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("No logger was provided"))
	})

	It("is an error when a parser returns neither a result nor an error", func() {
		results, err := parsing.Parse(testResults, []parsing.Parser{NeitherErrorNorResultsParser{}}, log)

		Expect(results).To(BeNil())
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("NeitherErrorNorResultsParser did not error and did not return a parse result"),
		)
	})

	It("is an error when no parsers can parse", func() {
		results, err := parsing.Parse(
			testResults,
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

	It("returns the first parse result with the highest sentiment", func() {
		results, err := parsing.Parse(
			testResults,
			[]parsing.Parser{
				NeutralSentimentParser{},
				NegativeSentimentParser{},
				ErrorParser{},
				PositiveSentimentParser{},
			},
			log,
		)

		Expect(results).NotTo(BeNil())
		Expect(*results.Framework.ProvidedKind).To(Equal("positive"))
		Expect(err).To(BeNil())

		logMessages := make([]string, 0)
		for _, log := range recordedLogs.All() {
			logMessages = append(logMessages, log.Message)
		}

		Expect(logMessages).To(ContainElement(
			ContainSubstring("ErrorParser was not capable of parsing the test results"),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("NeutralSentimentParser was capable of parsing the test results. Sentiment: Neutral"),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("PositiveSentimentParser was capable of parsing the test results. Sentiment: Positive"),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("NegativeSentimentParser was capable of parsing the test results. Sentiment: Negative"),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("PositiveSentimentParser was ultimately responsible for parsing the test results"),
		))
	})

	It("returns results with neutral sentiment", func() {
		results, err := parsing.Parse(
			testResults,
			[]parsing.Parser{
				NeutralSentimentParser{},
				ErrorParser{},
			},
			log,
		)

		Expect(results).NotTo(BeNil())
		Expect(*results.Framework.ProvidedKind).To(Equal("neutral"))
		Expect(err).To(BeNil())

		logMessages := make([]string, 0)
		for _, log := range recordedLogs.All() {
			logMessages = append(logMessages, log.Message)
		}

		Expect(logMessages).To(ContainElement(
			ContainSubstring("ErrorParser was not capable of parsing the test results"),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("NeutralSentimentParser was capable of parsing the test results. Sentiment: Neutral"),
		))
		Expect(logMessages).To(ContainElement(
			ContainSubstring("NeutralSentimentParser was ultimately responsible for parsing the test results"),
		))
	})

	It("returns an error if we only have negative sentiments", func() {
		results, err := parsing.Parse(
			testResults,
			[]parsing.Parser{
				NegativeSentimentParser{},
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
		Expect(logMessages).To(ContainElement(
			ContainSubstring("NegativeSentimentParser was capable of parsing the test results. Sentiment: Negative"),
		))
		Expect(logMessages).NotTo(ContainElement(
			ContainSubstring("ultimately responsible for parsing the test results"),
		))
	})
})
