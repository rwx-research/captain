// parsing holds the functionality to attempt to parse a test results file and the parsers themselves
// the parsers will produce a testingschema test result or an error
// in the case where no parsers are capable of parsing test results, an error will be returned indicating so
package parsing

import (
	"io"
	"sort"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

func Parse(testResults io.Reader, parsers []Parser, log *zap.SugaredLogger) (*v1.TestResults, error) {
	if len(parsers) == 0 {
		return nil, errors.NewInternalError("No parsers were provided")
	}

	if log == nil {
		return nil, errors.NewInternalError("No logger was provided")
	}

	parseResults := make([]ParseResult, 0)
	for _, parser := range parsers {
		parseResult, err := parser.Parse(testResults)
		if err != nil {
			log.Debugf("%T was not capable of parsing the test results. Error: %v", parser, err)
			continue
		}
		if parseResult == nil {
			return nil, errors.NewInternalError("%T did not error and did not return a parse result", parser)
		}
		log.Debugf("%T was capable of parsing the test results. Sentiment: %v", parser, parseResult.Sentiment)

		parseResults = append(parseResults, *parseResult)
	}

	if len(parseResults) == 0 {
		return nil, errors.NewInputError("No parsers were capable of parsing the provided test results")
	}

	sort.SliceStable(parseResults, func(i, j int) bool { return parseResults[i].Sentiment > parseResults[j].Sentiment })
	finalResults := parseResults[0]

	if finalResults.Sentiment < NeutralParseResultSentiment {
		return nil, errors.NewInputError("No parsers were capable of parsing the provided test results")
	}

	log.Debugf("%T was ultimately responsible for parsing the test results", finalResults.Parser)
	return &finalResults.TestResults, nil
}
