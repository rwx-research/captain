// parsing holds the functionality to attempt to parse a test results file and the parsers themselves
// the parsers will produce a testingschema test result or an error
// in the case where no parsers are capable of parsing test results, an error will be returned indicating so
package parsing

import (
	"encoding/base64"
	"io"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type Config struct {
	ProvidedFrameworkKind     string
	ProvidedFrameworkLanguage string
	MutuallyExclusiveParsers  []Parser
	GenericParsers            []Parser
	FrameworkParsers          map[v1.Framework][]Parser
	Logger                    *zap.SugaredLogger
}

func Parse(file fs.File, groupNumber int, cfg Config) (*v1.TestResults, error) {
	if cfg.Logger == nil {
		return nil, errors.NewInternalError("No logger was provided")
	}

	if (cfg.ProvidedFrameworkKind != "" && cfg.ProvidedFrameworkLanguage == "") ||
		(cfg.ProvidedFrameworkKind == "" && cfg.ProvidedFrameworkLanguage != "") {
		return nil, errors.NewInputError("Must provide both language and kind when one is provided")
	}

	var parsers []Parser
	var coercedFramework *v1.Framework
	if cfg.ProvidedFrameworkKind == "" && cfg.ProvidedFrameworkLanguage == "" {
		parsers = append(parsers, cfg.MutuallyExclusiveParsers...)
		parsers = append(parsers, cfg.GenericParsers...)
	} else {
		framework := v1.CoerceFramework(cfg.ProvidedFrameworkLanguage, cfg.ProvidedFrameworkKind)

		if framework.IsOther() {
			parsers = cfg.GenericParsers
		} else {
			parsers = append(parsers, cfg.FrameworkParsers[framework]...)
		}

		coercedFramework = &framework
	}

	results, err := parseWith(file, parsers, groupNumber, cfg.Logger)
	if err != nil {
		return nil, err
	}

	if coercedFramework != nil {
		results.Framework = *coercedFramework
	}

	if results.Framework.IsOther() && !results.Framework.IsProvided() {
		cfg.Logger.Warnf(
			"We could not determine which framework produced your test results. " +
				"For a better experience in Captain, specify both your --language and --framework.",
		)
	}

	return results, nil
}

func parseWith(file fs.File, parsers []Parser, groupNumber int, log *zap.SugaredLogger) (*v1.TestResults, error) {
	if len(parsers) == 0 {
		return nil, errors.NewInternalError("No parsers were provided")
	}

	parsedTestResults := make([]v1.TestResults, 0)
	var firstParser Parser
	for _, parser := range parsers {
		if err := rewindFile(file); err != nil {
			return nil, err
		}

		parsedTestResult, err := parser.Parse(file)
		if err != nil {
			log.Debugf("%T was not capable of parsing the test results. Error: %v", parser, err)
			continue
		}
		if parsedTestResult == nil {
			return nil, errors.NewInternalError("%T did not error and did not return a test result", parser)
		}
		log.Debugf("%T was capable of parsing the test results.", parser)

		if firstParser == nil {
			firstParser = parser
		}
		parsedTestResults = append(parsedTestResults, *parsedTestResult)
	}

	if len(parsedTestResults) == 0 {
		return nil, errors.NewInputError("No parsers were capable of parsing the provided test results")
	}

	finalResults := parsedTestResults[0]
	log.Debugf("%T was ultimately responsible for parsing the test results", firstParser)

	if err := rewindFile(file); err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.NewSystemError("Unable to read file into buffer: %s", err)
	}
	if err := rewindFile(file); err != nil {
		return nil, err
	}

	// only set DerivedFrom for non-RWX parsers
	if _, ok := firstParser.(RWXParser); !ok {
		finalResults.DerivedFrom = []v1.OriginalTestResults{
			{
				OriginalFilePath: file.Name(),
				Contents:         base64.StdEncoding.EncodeToString(buf),
				GroupNumber:      groupNumber,
			},
		}
	}

	return &finalResults, nil
}

func rewindFile(file fs.File) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.NewSystemError("Unable to rewind file: %s", err)
	}

	return nil
}
