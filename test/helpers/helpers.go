package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/onsi/gomega"
)

func ReadEnvFromFile(fileName string) map[string]string {
	// use shell to parse the env file to support quotations, comments, etc

	// #nosec G204 -- test where we we control the filename
	cmd := exec.Command("env", "-i", "bash", "-c", fmt.Sprintf("source %s && env", fileName))
	output, err := cmd.Output()

	// fmt.Fprintf(GinkgoWriter, "ENV OUTPUT: %s", string(output))

	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	readEnv := map[string]string{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.SplitN(line, "=", 2)
		if fields[0] == "_" || fields[0] == "SHLVL" || fields[0] == "PWD" || len(fields) != 2 {
			continue
		}
		readEnv[fields[0]] = fields[1]
	}
	return readEnv
}

func SetEnvFromFile(fileName string) {
	for name, value := range ReadEnvFromFile(fileName) {
		os.Setenv(name, value)
	}
}

func SetEnv(env map[string]string) {
	for name, value := range env {
		os.Setenv(name, value)
	}
}

func UnsetCIEnv() {
	envPrefixes := []string{"GITHUB", "BUILDKITE", "CIRCLE", "GITLAB", "CI", "RWX", "CAPTAIN"}
	for _, env := range os.Environ() {
		for _, prefix := range envPrefixes {
			if strings.HasPrefix(env, prefix) {
				pair := strings.SplitN(env, "=", 2)
				os.Unsetenv(pair[0])
			}
		}
	}
}
