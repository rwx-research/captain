package mint

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

const (
	retryActionKey         = "retry-failed-tests"
	retryActionLabel       = "Retry failed tests"
	retryActionDescription = "Only the tests that failed will be run again."
	retryActionEnv         = "CAPTAIN_MINT_RETRY_FAILED_TESTS"
)

func IsMint() bool {
	return os.Getenv("MINT") == "true"
}

func DidRetryFailedTests() bool {
	return os.Getenv(retryActionEnv) == "true"
}

func WriteConfigureRetryCommandTip(fs fs.FileSystem) error {
	var err error

	tipFile, err := fs.Create(filepath.Join(os.Getenv("MINT_TIPS"), "configure-captain-retry-command"))
	if err != nil {
		return errors.Wrap(err, "unable to create tip file")
	}
	defer tipFile.Close()

	_, err = tipFile.Write([]byte("Configure Captain retry command to enable Mint to retry only failed tests."))
	if err != nil {
		return errors.Wrap(err, "unable to write to tip file")
	}

	return nil
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

	err = writeDescription(fs)
	if err != nil {
		return errors.Wrap(err, "unable to write the Mint retry action description file")
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
		return nil, errors.Wrap(err, "unable to open test results file")
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
		return errors.Wrap(err, "unable to create error file")
	}
	defer errorFile.Close()

	_, err = errorFile.Write([]byte(errorMessage))
	if err != nil {
		return errors.Wrap(err, "unable to write to error file")
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
		return errors.Wrap(err, "unable to create env file")
	}
	defer envFile.Close()

	_, err = envFile.Write([]byte("true"))
	if err != nil {
		return errors.Wrap(err, "unable to write to env file")
	}

	return nil
}

func writeLabel(fs fs.FileSystem) error {
	var err error

	labelFile, err := fs.Create(filepath.Join(retryActionDirectory(), "label"))
	if err != nil {
		return errors.Wrap(err, "unable to create label file")
	}
	defer labelFile.Close()

	_, err = labelFile.Write([]byte(RetryFailedTestsLabel()))
	if err != nil {
		return errors.Wrap(err, "unable to write to label file")
	}

	return nil
}

func writeDescription(fs fs.FileSystem) error {
	var err error

	descriptionFile, err := fs.Create(filepath.Join(retryActionDirectory(), "description"))
	if err != nil {
		return errors.Wrap(err, "unable to create description file")
	}
	defer descriptionFile.Close()

	_, err = descriptionFile.Write([]byte(retryActionDescription))
	if err != nil {
		return errors.Wrap(err, "unable to write to description file")
	}

	return nil
}

func writeTestResults(fs fs.FileSystem, testResults v1.TestResults) error {
	var err error

	testResultsFile, err := fs.Create(filepath.Join(retryActionDirectory(), "data", "test-results.json"))
	if err != nil {
		return errors.Wrap(err, "unable to create test results file")
	}
	defer testResultsFile.Close()

	encoder := json.NewEncoder(testResultsFile)
	encoder.SetIndent("", "") // make them compact

	if err := encoder.Encode(testResults); err != nil {
		return errors.Wrap(err, "unable to encode test results to JSON")
	}

	return nil
}
