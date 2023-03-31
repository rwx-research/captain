//go:build integration

package integration_test

import (
	"bytes"
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

func mergeMaps(into map[string]string, from map[string]string) map[string]string {
	for key, value := range from {
		into[key] = value
	}

	return into
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

func withAndWithoutInheritedEnv(sharedTests sharedTestGen) {
	if os.Getenv("CI") != "" {
		// these tests are only run in CI
		Describe("using CI environment", func() {
			sharedTests(envAsMap, "inherited-env")
		})
	}

	if os.Getenv("ONLY_TEST_INHERITED_ENV") == "" {
		// don't run these on buildkite
		Describe("using the generic provider", func() {
			sharedTests(func() map[string]string {
				return helpers.ReadEnvFromFile(".env.captain")
			}, "generic")
		})

		if os.Getenv("FAST_INTEGRATION") == "" {
			Describe("using the GitHub Actions provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.github.actions")
				}, "github-actions")
			})

			Describe("using the GitLab CI provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.gitlab_ci")
				}, "gitlab-ci")
			})

			Describe("using the CircleCI provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.circleci")
				}, "circleci")
			})

			Describe("using the Buildkite provider", func() {
				sharedTests(func() map[string]string {
					return helpers.ReadEnvFromFile(".env.buildkite")
				}, "buildkite")
			})
		}
	}
}

type envGenerator func() map[string]string

type sharedTestGen func(envGenerator, string)
