//go:build integration

package integration_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
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
