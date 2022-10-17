//go:build mage
// +build mage

package main

import (
	"context"
	"os/exec"

	"github.com/magefile/mage/sh"
)

// Default is the default build target.
var Default = Build

// Build builds the Captain CLI
func Build(ctx context.Context) error {
	return sh.RunV("go", "build", "./cmd/captain")
}

// Clean removes any generated artifacts from the repository.
func Clean(ctx context.Context) error {
	return sh.Rm("./captain")
}

// Lint runs the linter & performs static-analysis checks.
func Lint(ctx context.Context) error {
	return sh.RunV("golangci-lint", "run", "./...")
}

// Test executes the test-suite for the Captain-CLI.
func Test(ctx context.Context) error {
	cmd := exec.Command("command", "-v", "ginkgo")
	if err := cmd.Run(); err != nil {
		return sh.RunV("go", "test", "./...")
	}

	return sh.RunV("ginkgo", "./...")
}
