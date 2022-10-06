//go:build mage
// +build mage

package main

import (
	"context"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default is the default build target.
var Default = Build

// Build creates a production-ready build of the Captain CLI.
func Build(ctx context.Context) error {
	mg.Deps(Test)

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

// Run runs the Captain CLI in a developer mode.
func Run(ctx context.Context) error {
	return sh.RunV("go", "run", "./cmd/captain")
}

// Test executes the test-suite for the Captain-CLI.
func Test(ctx context.Context) error {
	return sh.RunV("go", "test", "./...")
}
