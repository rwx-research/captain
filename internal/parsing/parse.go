// parsing holds the functionality to attempt to parse a test results file and the parsers themselves
// the parsers will produce a testingschema test result or an error
// in the case where no parsers are capable of parsing test results, an error will be returned indicating so
package parsing

import (
	"encoding/base64"
	"fmt"
	"io"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type Config struct {
	ProvidedFrameworkKind     string
	ProvidedFrameworkLanguage string
	FailOnDuplicateTestID     bool
	MutuallyExclusiveParsers  []Parser
	GenericParsers            []Parser
	FrameworkParsers          map[v1.Framework][]Parser
	IdentityRecipes           map[string]v1.TestIdentityRecipe
	Logger                    *zap.SugaredLogger
}

func (c Config) Validate() error {
	if c.ProvidedFrameworkKind != "" && c.ProvidedFrameworkLanguage == "" {
		return errors.NewConfigurationError(
			"Unable to determine test result format",
			"You provided the test framework that produced the test result, but not the language. Captain needs "+
				"both options in order to interpret the results file correctly.",
			"The test-framework language can be set using the --language flag. Alternatively, you can use "+
				"the Captain configuration file to permanently set the framework options for a test suite.",
		)
	}

	if c.ProvidedFrameworkLanguage != "" && c.ProvidedFrameworkKind == "" {
		return errors.NewConfigurationError(
			"Unable to determine test result format",
			"You provided the test-framework language that produced the test result, but not the framework name "+
				"itself. Captain needs both options in order to interpret the results file correctly.",
			"The test framework can be set using the --framework flag. Alternatively, you can use "+
				"the Captain configuration file to permanently set the framework options for a test suite.",
		)
	}

	if c.Logger == nil {
		return errors.NewInternalError("No logger was provided")
	}

	return nil
}

func Parse(file fs.File, groupNumber int, cfg Config) (*v1.TestResults, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.WithStack(err)
	}

	var parsers []Parser
	var coercedFramework *v1.Framework
	if cfg.ProvidedFrameworkKind == "" && cfg.ProvidedFrameworkLanguage == "" {
		parsers = append(parsers, cfg.MutuallyExclusiveParsers...)
		parsers = append(parsers, cfg.GenericParsers...)
	} else {
		framework := v1.CoerceFramework(cfg.ProvidedFrameworkLanguage, cfg.ProvidedFrameworkKind)

		if framework.IsOther() {
			cfg.Logger.Warnf(
				"Could not find a suitable parser for %q and %q - Captain will fall back to generic parsers.",
				cfg.ProvidedFrameworkLanguage,
				cfg.ProvidedFrameworkKind,
			)
			cfg.Logger.Warnln(
				"If this fails, omit the `--language` and `--framework` flags to attempt parsing with all available parsers" +
					" instead.",
			)
			parsers = cfg.GenericParsers
		} else {
			parsers = append(parsers, cfg.FrameworkParsers[framework]...)
		}

		coercedFramework = &framework
	}

	results, err := parseWith(file, parsers, groupNumber, cfg)
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

func parseWith(file fs.File, parsers []Parser, groupNumber int, cfg Config) (*v1.TestResults, error) {
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
			cfg.Logger.Debugf("%T was not capable of parsing the test results. Error: %v", parser, err)
			continue
		}
		if parsedTestResult == nil {
			return nil, errors.NewInternalError("%T did not error and did not return a test result", parser)
		}

		cfg.Logger.Debugf("%T was capable of parsing the test results.", parser)

		if firstParser == nil {
			firstParser = parser
		}
		parsedTestResults = append(parsedTestResults, *parsedTestResult)
	}

	if len(parsedTestResults) == 0 {
		return nil, errors.NewInputError("No parsers were capable of parsing the provided test results")
	}

	finalResults := parsedTestResults[0]
	cfg.Logger.Debugf("%T was ultimately responsible for parsing the test results", firstParser)

	testIDsAreUnique, err := checkIfTestIDsAreUnique(finalResults, cfg)
	if err != nil {
		return nil, err
	}
	if !testIDsAreUnique {
		duplicateTestIDWarningMsg := fmt.Sprintf(
			"Test result file %q contains two or more tests that share the same metadata (such as name or identifier). %s",
			file.Name(),
			"Please make sure test identifiers are unique.",
		)
		if cfg.FailOnDuplicateTestID {
			return nil, errors.NewInputError(duplicateTestIDWarningMsg)
		}
		cfg.Logger.Warn(duplicateTestIDWarningMsg)
	}

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

func checkIfTestIDsAreUnique(testResult v1.TestResults, cfg Config) (bool, error) {
	uniqueTestIdentifiers := make(map[string]struct{})
	identityRecipe, recipeFound := cfg.IdentityRecipes[testResult.Framework.String()]

	// Populate `uniqueTestIdentifiers` with the ID of the last test as the for-loop that follows will
	// not cover it.
	if recipeFound {
		id, err := testResult.Tests[len(testResult.Tests)-1].Identify(identityRecipe)
		if err != nil {
			cfg.Logger.Warnf("Unable to construct identity from test: %s", err.Error())
		} else {
			uniqueTestIdentifiers[id] = struct{}{}
		}
	}

	for i := 0; i < len(testResult.Tests)-1; i++ {
		test := testResult.Tests[i]

		if recipeFound {
			id, err := test.Identify(identityRecipe)
			if err != nil {
				cfg.Logger.Warnf("Unable to construct identity from test: %s", err.Error())
			} else {
				if _, ok := uniqueTestIdentifiers[id]; ok {
					return false, nil
				}
				uniqueTestIdentifiers[id] = struct{}{}
			}
		}

		for j := i + 1; j < len(testResult.Tests); j++ {
			if test.Matches(testResult.Tests[j]) {
				return false, nil
			}
		}
	}

	return true, nil
}

func rewindFile(file fs.File) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.NewSystemError("Unable to rewind file: %s", err)
	}

	return nil
}
