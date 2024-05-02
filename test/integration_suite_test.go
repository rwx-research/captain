//go:build integration

package integration_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rwx-research/captain-cli/test/helpers"
)

func TestCaptainBinary(t *testing.T) {
	t.Parallel()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = SynchronizedAfterSuite(
	func() {}, // do nothing on all processes
	func() { // on the first process, delete the captain dir
		err := os.RemoveAll(".captain")
		Expect(err).ToNot(HaveOccurred())
	})

// helpers
type captainArgs struct {
	args []string
	env  map[string]string // note: we always set PATH
}

type captainResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func print(output string) {
	fmt.Fprintf(GinkgoWriter, output)
}

// used to debug tests
func printResults(result captainResult) {
	print(fmt.Sprintf("captain stdout: %s, captain stderr: %s, captain exit code: %d", result.stdout, result.stderr, result.exitCode))
}

func captainCmd(args captainArgs) *exec.Cmd {
	const captainPath = "../captain"
	_, err := os.Stat(captainPath)
	Expect(err).ToNot(HaveOccurred(), "integration tests depend on a captain binary in the root directory. This should be created automatically with `mage integrationTest` and `mage test`")

	cmd := exec.Command("../captain", args.args...)

	env := []string{
		fmt.Sprintf("%s=%s", "PATH", os.Getenv("PATH")),
	}

	for key, value := range args.env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Env = env

	fmt.Fprintf(GinkgoWriter, "Executing command: %s\n with env %s\n", cmd.String(), cmd.Env)

	return cmd
}

func runCaptain(args captainArgs) captainResult {
	cmd := captainCmd(args)
	var stdoutBuffer, stderrBuffer bytes.Buffer

	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err := cmd.Run()

	exitCode := 0

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		Expect(ok).To(BeTrue(), "captain exited with an error that wasn't an ExitError")
		exitCode = exitErr.ExitCode()
	}

	return captainResult{
		stdout:   strings.TrimSuffix(stdoutBuffer.String(), "\n"),
		stderr:   strings.TrimSuffix(stderrBuffer.String(), "\n"),
		exitCode: exitCode,
	}
}

func envAsMap() map[string]string {
	env := os.Environ()
	result := map[string]string{}

	for _, envVar := range env {
		parts := strings.SplitN(envVar, "=", 2)
		result[parts[0]] = parts[1]
	}

	return result
}

// this is git-sha-ish. We could throw the `sha` function in between
// but a random hex string should be good enough
func randomGitSha() string {
	b := make([]byte, 20)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func copyMap(m map[string]string) map[string]string {
	result := map[string]string{}

	for key, value := range m {
		result[key] = value
	}

	return result
}

func withAndWithoutInheritedEnv(sharedTests sharedTestGen) {
	// Intuitively I think this should be static per test run but I'm not sure if that's actually helpful or true
	randomGitSha := randomGitSha()

	if os.Getenv("CI") != "" {
		env := envAsMap()
		// don't overwrite the commit sha, but let's get rid of RWX_ACCESS_TOKEN
		delete(env, "RWX_ACCESS_TOKEN")

		// these tests are only run in CI
		Describe("using CI environment", func() {
			sharedTests(func() map[string]string { return copyMap(env) }, "inherited-env")
		})
	}

	if os.Getenv("ONLY_TEST_INHERITED_ENV") == "" {
		// don't run these on buildkite
		Describe("using the generic provider", func() {
			env := helpers.ReadEnvFromFile(".env.captain")
			env["CAPTAIN_SHA"] = randomGitSha

			sharedTests(func() map[string]string {
				return copyMap(env)
			}, "generic")
		})

		if os.Getenv("FAST_INTEGRATION") == "" {
			Describe("using the Mint provider", func() {
				env := helpers.ReadEnvFromFile(".env.mint")
				env["MINT_GIT_COMMIT_SHA"] = randomGitSha
				sharedTests(func() map[string]string {
					return copyMap(env)
				}, "mint")
			})

			Describe("using the GitHub Actions provider", func() {
				env := helpers.ReadEnvFromFile(".env.github.actions")
				env["GITHUB_SHA"] = randomGitSha
				sharedTests(func() map[string]string {
					return copyMap(env)
				}, "github-actions")
			})

			Describe("using the GitLab CI provider", func() {
				env := helpers.ReadEnvFromFile(".env.gitlab_ci")
				env["CI_COMMIT_SHA"] = randomGitSha
				sharedTests(func() map[string]string {
					return copyMap(env)
				}, "gitlab-ci")
			})

			Describe("using the CircleCI provider", func() {
				env := helpers.ReadEnvFromFile(".env.circleci")
				env["CIRCLE_SHA1"] = randomGitSha
				sharedTests(func() map[string]string {
					return copyMap(env)
				}, "circleci")
			})

			Describe("using the Buildkite provider", func() {
				env := helpers.ReadEnvFromFile(".env.buildkite")
				env["BUILDKITE_COMMIT"] = randomGitSha
				sharedTests(func() map[string]string {
					return copyMap(env)
				}, "buildkite")
			})
		}
	}
}

type envGenerator func() map[string]string

type sharedTestGen func(envGenerator, string)

func withCaptainConfig(cfg string, rootDir string, body func(configPath string)) {
	// TODO set RootDir on the config file once we get that merged
	configPath := filepath.Join(rootDir, ".captain/config.yaml")

	err := os.MkdirAll(filepath.Dir(configPath), 0o750)
	Expect(err).NotTo(HaveOccurred())

	err = os.WriteFile(configPath, []byte(cfg), 0o600)
	Expect(err).NotTo(HaveOccurred())

	defer func() {
		err := os.Remove(configPath)
		Expect(err).NotTo(HaveOccurred())
	}()

	body(configPath)
}

// Integration tests are run to ensure that captain behaves as expected as well as to ensure background compatibility
// They're here to ensure
// - old CLI args should work
// - old ENV vars should work
// - old config files should work
// Some guidelines for writing them:
// - avoid asserting on STDERR
// - assertions should be as lenient as possible (e.g. avoid asserting on exact values when it's unnecessary to guarantee compatibility)
//
// If you want to write tighter assertions that you expect to break slightly future versions, wrap those assertions
// (or the whole tests) in `withoutBackwardsCompatibility`
func withoutBackwardsCompatibility(incompatibleClosure func()) {
	if os.Getenv("LEGACY_VERSION_TO_TEST") == "" {
		incompatibleClosure()
	}
}

// we add a prefix to top level describes to allow quarantining (disabling) of legacy tests that become invalid / are subject to a breaking change
func versionedPrefixForQuarantining() string {
	version := os.Getenv("LEGACY_VERSION_TO_TEST")
	if os.Getenv("LEGACY_VERSION_TO_TEST") == "" {
		return ""
	} else {
		return version + ": "
	}
}
