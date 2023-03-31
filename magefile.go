//go:build mage
// +build mage

package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
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
		UnitTest,
		IntegrationTest,
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
	if err := sh.RunV("golangci-lint", "run", "--fix", "./..."); err != nil {
		return err
	}

	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return err
	}

	return nil
}

func Test(ctx context.Context) error {
	mg.Deps(Build)
	return (makeTestTask("-tags", "integration", "./..."))(ctx)
}

// Test executes the test-suite for the Captain-CLI.
func UnitTest(ctx context.Context) error {
	// `ginkgo ./...` or `go test ./...`  work out of the box
	// but `ginkgo ./...` includes ~ confusing empty test output for integration tests
	// so `mage test` explicitly doesn't call ginkgo against the `/test/` directory
	return (makeTestTask("./internal/...", "./cmd/..."))(ctx)
}

// Test executes the test-suite for the Captain-CLI.
func IntegrationTest(ctx context.Context) error {
	mg.Deps(Build)
	return (makeTestTask("-tags", "integration", "./test/"))(ctx)
}

func makeTestTask(args ...string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if report := os.Getenv("REPORT"); report != "" {
			return sh.RunV("ginkgo", append([]string{"-p", "--junit-report=report.xml"}, args...)...)
		}

		cmd := exec.Command("command", "-v", "ginkgo")
		if err := cmd.Run(); err != nil {
			return sh.RunV("go", append([]string{"test"}, args...)...)
		}

		return sh.RunV("ginkgo", append([]string{"-p"}, args...)...)
	}
}
