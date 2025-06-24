//go:build mage
// +build mage

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/blang/semver/v4"
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
		LegacyTestSuiteTags,
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

	ldflags, err := getLdflags()
	if err != nil {
		return err
	}
	args = append([]string{"-ldflags", ldflags}, args...)

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
	if os.Getenv("LEGACY_VERSION_TO_TEST") == "" {
		// only build captain if we're running tests against the latest captain version
		// so that we can build against the active, branch, then run integration tests from a previous commit against that
		// binary
		mg.Deps(Build)
	}

	return (makeTestTask("-tags", "integration", "./test/"))(ctx)
}

func makeTestTask(args ...string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		ldflags, err := getLdflags()
		if err != nil {
			return err
		}
		args = append([]string{"-ldflags", ldflags}, args...)

		// Check if this is an integration test run
		isIntegrationTest := false
		for _, arg := range args {
			if arg == "integration" || arg == "./test/" {
				isIntegrationTest = true
				break
			}
		}

		if report := os.Getenv("REPORT"); report != "" {
			if isIntegrationTest {
				return sh.RunV("ginkgo", append([]string{"--junit-report=report.xml"}, args...)...)
			} else {
				return sh.RunV("ginkgo", append([]string{"-p", "--junit-report=report.xml"}, args...)...)
			}
		}

		cmd := exec.Command("command", "-v", "ginkgo")
		if err := cmd.Run(); err != nil {
			return sh.RunV("go", append([]string{"test"}, args...)...)
		}

		if isIntegrationTest {
			return sh.RunV("ginkgo", args...)
		} else {
			return sh.RunV("ginkgo", append([]string{"-p"}, args...)...)
		}
	}
}

// this prints out a json-formatted list of tags where the ./test directory has changed
func LegacyTestSuiteTags(ctx context.Context) error {
	tagsWithChanges, err := findTagsWithChangesInTest()
	if err != nil {
		return err
	}

	output, err := json.Marshal(tagsWithChanges)
	if err != nil {
		return err
	}

	fmt.Println(string(output))
	return nil
}

// this prints out a json-formatted list of tags where the ./test directory has changed
func findTagsWithChangesInTest() ([]string, error) {
	out, err := exec.Command("git", "tag", "--sort=version:refname").Output()
	if err != nil {
		return nil, err
	}

	var versionTags []semver.Version

	for _, tag := range strings.Split(string(out), "\n") {
		tagVersion, err := semver.ParseTolerant(tag)
		if err == nil && // only include the tag if it parses...
			len(tagVersion.Pre) == 0 && // and is not pre-release...
			tagVersion.GTE(semver.Version{Major: 1, Minor: 15, Patch: 0}) { // and is >= 1.17.4

			versionTags = append(versionTags, tagVersion)
		}
	}

	// sort by version
	sort.Sort(semver.Versions(versionTags))

	// keep tags where there has been a change in ./test
	tagsWithChanges := []string{}

	for i := 0; i < len(versionTags)-1; i++ {
		prevVersion := versionTags[i]
		version := versionTags[i+1]
		hasChanges, err := hasChangesIn(prevVersion, version, "./test")
		if err != nil {
			return nil, err
		}

		if hasChanges {
			tagsWithChanges = append(tagsWithChanges, toTag(version))
		}
	}

	return tagsWithChanges, nil
}

func toTag(tag semver.Version) string {
	return fmt.Sprintf("v%s", tag.String())
}

func hasChangesIn(prevVersion semver.Version, version semver.Version, folder string) (bool, error) {
	err := exec.Command("git", "diff", "--quiet", toTag(prevVersion), toTag(version), "--", folder).Run()
	if err == nil {
		return false, nil
	}

	if _, ok := err.(*exec.ExitError); ok {
		return true, nil
	}

	return false, err
}

func getLdflags() (string, error) {
	if ldflags := os.Getenv("LDFLAGS"); ldflags != "" {
		return ldflags, nil
	}

	sha, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("-X github.com/rwx-research/captain-cli.Version=testing-%v", string(sha)), nil
}
