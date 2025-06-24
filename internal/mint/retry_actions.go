package mint

import (
	"encoding/json"
	"io"
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

func WriteIntermediateArtifacts(fs fs.FileSystem, intermediateArtifactsPath string) error {
	if intermediateArtifactsPath == "" {
		return nil
	}

	// Check if intermediate artifacts directory exists
	info, err := fs.Stat(intermediateArtifactsPath)
	if os.IsNotExist(err) {
		return nil // No artifacts to copy
	}
	if err != nil {
		return errors.Wrap(err, "unable to check intermediate artifacts directory")
	}
	if !info.IsDir() {
		return nil // Not a directory, nothing to copy
	}

	// Copy the entire intermediate artifacts directory to the Mint retry action data directory
	srcPath := intermediateArtifactsPath
	dstPath := filepath.Join(retryActionDirectory(), "data", "intermediate-artifacts")

	err = copyDirectory(fs, srcPath, dstPath)
	if err != nil {
		return errors.Wrap(err, "unable to copy intermediate artifacts to Mint retry action directory")
	}

	return nil
}

func copyDirectory(fs fs.FileSystem, src, dst string) error {
	// Create destination directory
	err := fs.MkdirAll(dst, 0o750)
	if err != nil {
		return errors.WithStack(err)
	}

	// Use glob to find all files recursively
	pattern := filepath.Join(src, "**", "*")
	allPaths, err := fs.Glob(pattern)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, srcPath := range allPaths {
		// Calculate relative path and destination
		relPath, err := filepath.Rel(src, srcPath)
		if err != nil {
			return errors.WithStack(err)
		}
		dstPath := filepath.Join(dst, relPath)

		// Check if it's a directory or file
		info, err := fs.Stat(srcPath)
		if err != nil {
			return errors.WithStack(err)
		}

		if info.IsDir() {
			// Create directory
			err = fs.MkdirAll(dstPath, 0o750)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			// Ensure parent directory exists
			parentDir := filepath.Dir(dstPath)
			err = fs.MkdirAll(parentDir, 0o750)
			if err != nil {
				return errors.WithStack(err)
			}

			// Copy file
			err = copyFile(fs, srcPath, dstPath)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

func copyFile(fs fs.FileSystem, src, dst string) error {
	srcFile, err := fs.Open(src)
	if err != nil {
		return errors.WithStack(err)
	}
	defer srcFile.Close()

	dstFile, err := fs.Create(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(dstFile.Sync())
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

func RestoreIntermediateArtifacts(fs fs.FileSystem, intermediateArtifactsPath string) error {
	if intermediateArtifactsPath == "" {
		return nil
	}

	// Check if intermediate artifacts exist in the mint retry data directory
	srcPath := filepath.Join(retryDataDirectory(), "intermediate-artifacts")
	info, err := fs.Stat(srcPath)
	if os.IsNotExist(err) {
		return nil // No artifacts to restore
	}
	if err != nil {
		return errors.Wrap(err, "unable to check mint retry data artifacts directory")
	}
	if !info.IsDir() {
		return nil // Not a directory, nothing to restore
	}

	// Copy the artifacts from mint retry data directory to the intermediate artifacts path
	err = copyDirectory(fs, srcPath, intermediateArtifactsPath)
	if err != nil {
		return errors.Wrap(err, "unable to restore intermediate artifacts from mint retry data")
	}

	return nil
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
