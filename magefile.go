//go:build mage
// +build mage

package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/magefile/mage/sh"
)

// Default is the default build target.
var Default = Build

// All cleans output, builds, tests, and lints.
func All(ctx context.Context) error {
	type target func(context.Context) error

	targets := []target{
		Clean,
		Build,
		Test,
		Lint,
		LintFix,
	}

	for _, t := range targets {
		if err := t(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Build builds the Captain CLI
func Build(ctx context.Context) error {
	args := []string{"./cmd/captain"}

	if ldflags := os.Getenv("LDFLAGS"); ldflags != "" {
		args = append([]string{"-ldflags", ldflags}, args...)
	}

	if cgo_enabled := os.Getenv("CGO_ENABLED"); cgo_enabled == "0" {
		args = append([]string{"-a"}, args...)
	}

	return sh.RunV("go", append([]string{"build"}, args...)...)
}

// Clean removes any generated artifacts from the repository.
func Clean(ctx context.Context) error {
	return sh.Rm("./captain")
}

// Lint runs the linter & performs static-analysis checks.
func Lint(ctx context.Context) error {
	return sh.RunV("golangci-lint", "run", "./...")
}

// Applies lint checks and fixes any issues.
func LintFix(ctx context.Context) error {
	return sh.RunV("golangci-lint", "run", "--fix", "./...")
}

// Test executes the test-suite for the Captain-CLI.
func Test(ctx context.Context) error {
	if report := os.Getenv("REPORT"); report != "" {
		return sh.RunV("ginkgo", "--junit-report=report.xml", "./...")
	}

	cmd := exec.Command("command", "-v", "ginkgo")
	if err := cmd.Run(); err != nil {
		return sh.RunV("go", "test", "./...")
	}

	return sh.RunV("ginkgo", "./...")
}
