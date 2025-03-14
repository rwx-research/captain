// parsing holds the functionality to attempt to parse a test results file and the parsers themselves
// the parsers will produce a testingschema test result or an error
// in the case where no parsers are capable of parsing test results, an error will be returned indicating so
package parsing

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

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

	duplicateTestIDs, err := checkIfTestIDsAreUnique(finalResults, cfg)
	if err != nil {
		return nil, err
	}
	if len(duplicateTestIDs) != 0 {
		duplicateTestIDWarningMsg := fmt.Sprintf(
			"Test result file %q contains two or more tests that share the same metadata:\n\n\t%s\n\n%s",
			file.Name(),
			strings.Join(duplicateTestIDs, "\n\t"),
			"Please make sure test identifiers are unique.",
		)
		if cfg.FailOnDuplicateTestID {
			return nil, errors.NewDuplicateTestIDError("%s", duplicateTestIDWarningMsg)
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

func checkIfTestIDsAreUnique(testResult v1.TestResults, cfg Config) ([]string, error) {
	uniqueTestIdentifiers := make(map[string]struct{})
	uniqueTestMatchingIdentities := make(map[string]struct{})
	duplicateTestIDs := make([]string, 0)
	identityRecipe, recipeFound := cfg.IdentityRecipes[testResult.Framework.String()]

	if !recipeFound && (cfg.ProvidedFrameworkKind != "" || cfg.ProvidedFrameworkLanguage != "") {
		identityRecipe, recipeFound = cfg.IdentityRecipes[v1.CoerceFramework(
			string(v1.FrameworkLanguageOther),
			string(v1.FrameworkKindOther),
		).String()]
	}

	for _, test := range testResult.Tests {
		var id string
		var err error

		// Check the identity based on the identity recipe for duplicates
		if recipeFound {
			id, err = test.Identify(identityRecipe)
			if err != nil {
				cfg.Logger.Warnf("Unable to construct identity from test: %s", err.Error())
			} else {
				if _, ok := uniqueTestIdentifiers[id]; ok {
					duplicateTestIDs = append(duplicateTestIDs, id)
				} else {
					uniqueTestIdentifiers[id] = struct{}{}
				}
			}
		}

		// Check the identity used for matching for duplicates
		identityForMatching := test.IdentityForMatching()
		if _, ok := uniqueTestMatchingIdentities[identityForMatching]; ok {
			if id != "" {
				duplicateTestIDs = append(duplicateTestIDs, id)
			} else {
				duplicateTestIDs = append(duplicateTestIDs, test.Name)
			}
		} else {
			uniqueTestMatchingIdentities[identityForMatching] = struct{}{}
		}
	}

	return duplicateTestIDs, nil
}

func rewindFile(file fs.File) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.NewSystemError("Unable to rewind file: %s", err)
	}

	return nil
}
