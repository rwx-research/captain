package mint

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

const retryActionKey = "retry-failed-tests"
const retryActionLabel = "Retry failed tests"
const retryActionEnv = "CAPTAIN_MINT_RETRY_FAILED_TESTS"

func IsMint() bool {
	return os.Getenv("MINT") == "true"
}

func DidRetryFailedTests() bool {
	return os.Getenv(retryActionEnv) == "true"
}

func WriteRetryFailedTestsAction(fs fs.FileSystem, testResults v1.TestResults) error {
	var err error

	err = writeEnv(fs)
	if err != nil {
		return errors.Wrap(err, "unable to write the Mint retry action env file")
	}

	err = writeLabel(fs)
	if err != nil {
		return errors.Wrap(err, "unable to write the Mint retry action label file")
	}

	err = writeTestResults(fs, testResults)
	if err != nil {
		return errors.Wrap(err, "unable to write the Mint retry action test results file")
	}

	return nil
}

func ReadFailedTestResults(fs fs.FileSystem) (*v1.TestResults, error) {
	var err error

	testResultsFile, err := fs.Open(filepath.Join(retryDataDirectory(), "test-results.json"))
	if err != nil {
		return nil, err
	}
	defer testResultsFile.Close()

	var testResults v1.TestResults
	err = json.NewDecoder(testResultsFile).Decode(&testResults)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode test results from JSON")
	}

	return &testResults, nil
}

func WriteError(fs fs.FileSystem, errorMessage string) error {
	var err error

	errorFile, err := fs.CreateTemp(errorsDirectory(), "captain-")
	if err != nil {
		return err
	}
	defer errorFile.Close()

	_, err = errorFile.Write([]byte(errorMessage))
	if err != nil {
		return err
	}

	return nil
}

func RetryFailedTestsLabel() string {
	return retryActionLabel
}

func errorsDirectory() string {
	return os.Getenv("MINT_ERRORS")
}

func retryDataDirectory() string {
	return os.Getenv("MINT_RETRY_DATA")
}

func retryActionDirectory() string {
	return filepath.Join(os.Getenv("MINT_RETRY_ACTIONS"), retryActionKey)
}

func writeEnv(fs fs.FileSystem) error {
	var err error

	envFile, err := fs.Create(filepath.Join(retryActionDirectory(), "env", retryActionEnv))
	if err != nil {
		return err
	}
	defer envFile.Close()

	_, err = envFile.Write([]byte("true"))
	if err != nil {
		return err
	}

	return nil
}

func writeLabel(fs fs.FileSystem) error {
	var err error

	labelFile, err := fs.Create(filepath.Join(retryActionDirectory(), "label"))
	if err != nil {
		return err
	}
	defer labelFile.Close()

	_, err = labelFile.Write([]byte(RetryFailedTestsLabel()))
	if err != nil {
		return err
	}

	return nil
}

func writeTestResults(fs fs.FileSystem, testResults v1.TestResults) error {
	var err error

	testResultsFile, err := fs.Create(filepath.Join(retryActionDirectory(), "data", "test-results.json"))
	if err != nil {
		return err
	}
	defer testResultsFile.Close()

	encoder := json.NewEncoder(testResultsFile)
	encoder.SetIndent("", "") // make them compact

	if err := encoder.Encode(testResults); err != nil {
		return errors.Wrap(err, "unable to encode test results to JSON")
	}

	return nil
}
