//go:build integration

package integration_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// generateRandomSuffix generates a random 10-character hex string
func generateRandomSuffix() string {
	bytes := make([]byte, 5)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// createUniqueFile creates a unique copy of the source file
// Returns the unique path and a cleanup function
func createUniqueFile(srcPath string, prefix string) (string, func()) {
	randomSuffix := generateRandomSuffix()
	uniqueFileName := fmt.Sprintf("%s-%s-UNIQUE-%s", prefix, filepath.Base(srcPath), randomSuffix)
	uniqueFilePath := filepath.Join("fixtures/integration-tests", uniqueFileName)

	copyFile(srcPath, uniqueFilePath)

	cleanup := func() {
		os.Remove(uniqueFilePath)
	}

	return uniqueFilePath, cleanup
}

func copyFile(srcPath, dstPath string) error {
	// Open the source file for reading
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close() // Ensure the source file is closed

	// Create the destination file for writing
	// os.Create truncates the file if it already exists
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close() // Ensure the destination file is closed

	// Copy the contents from source to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Ensure all buffered data is written to the destination file
	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}
