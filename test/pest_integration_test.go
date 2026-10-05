//go:build pest

package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/gomega"
)

func TestPest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	captain, err := filepath.Abs("../captain")
	g.Expect(err).NotTo(HaveOccurred())
	vendor, err := filepath.Abs("pest/fixture/vendor")
	g.Expect(err).NotTo(HaveOccurred())

	prepare := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		g.Expect(os.CopyFS(filepath.Join(dir, "tests"), os.DirFS("pest/fixture/tests"))).To(Succeed())
		config, err := os.ReadFile("pest/fixture/phpunit.xml")
		g.Expect(err).NotTo(HaveOccurred())
		// #nosec G703 -- dir is created by testing.T.TempDir, not user input.
		g.Expect(os.WriteFile(filepath.Join(dir, "phpunit.xml"), config, 0o600)).To(Succeed())
		g.Expect(os.CopyFS(filepath.Join(dir, "vendor"), os.DirFS(vendor))).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(dir, "captain.yaml"), []byte(`cloud:
  disabled: true
test-suites:
  pest:
    command: vendor/bin/pest --log-junit results.xml
    results:
      language: PHP
      framework: Pest
      path: results.xml
    retries:
      attempts: 1
      command: vendor/bin/pest --log-junit results.xml --filter '{{ filter }}' '{{ file }}'
    partition:
      globs: [tests/*Test.php]
      command: vendor/bin/pest --log-junit results.xml {{ testFiles }}
`), 0o600)).To(Succeed())
		return dir
	}
	run := func(dir string, args ...string) []byte {
		// #nosec G204 -- execute the locally built Captain with arguments owned by this test.
		cmd := exec.CommandContext(t.Context(), captain, append([]string{"--config-file", "captain.yaml"}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "CAPTAIN_PEST_ATTEMPTS="+filepath.Join(dir, "attempts.json"))
		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred(), string(output))
		return output
	}
	readResults := func(dir string) v1.TestResults {
		data, err := os.ReadFile(filepath.Join(dir, "report.json"))
		g.Expect(err).NotTo(HaveOccurred())
		var results v1.TestResults
		g.Expect(json.Unmarshal(data, &results)).To(Succeed())
		return results
	}

	dir := prepare(t)
	run(dir, "run", "pest", "--update-stored-results", "--reporter", "rwx-v1-json=report.json")
	results := readResults(dir)
	g.Expect(results.Framework).To(Equal(v1.PHPPestFramework))
	g.Expect(results.Tests).To(HaveLen(10))
	expectedTimings := make(map[string]time.Duration)
	retried := 0
	for _, test := range results.Tests {
		g.Expect(test.Location.File).To(HaveSuffix(".php"))
		g.Expect(test.Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful), test.Name)
		expectedTimings[test.Location.File] += *test.Attempt.Duration
		if len(test.PastAttempts) > 0 {
			retried++
		}
	}
	g.Expect(retried).To(BeNumerically(">=", 5))
	timingsData, err := os.ReadFile(filepath.Join(dir, ".captain/pest/timings.yaml"))
	g.Expect(err).NotTo(HaveOccurred())
	var timings map[string]time.Duration
	g.Expect(yaml.Unmarshal(timingsData, &timings)).To(Succeed())
	g.Expect(timings).To(Equal(expectedTimings))
	data, err := os.ReadFile(filepath.Join(dir, "attempts.json"))
	g.Expect(err).NotTo(HaveOccurred())
	var attempts map[string]int
	g.Expect(json.Unmarshal(data, &attempts)).To(Succeed())
	datasetAttempts := 2
	if strings.HasPrefix(os.Getenv("PEST_VERSION"), "4.") {
		datasetAttempts = 1
	}
	g.Expect(attempts).To(Equal(map[string]int{
		"punctuation": 2, "similar": 1, "nested": 2, "named-failed": 2, "named-passed": datasetAttempts,
		"numbered-failed": 2, "numbered-passed": datasetAttempts, "other": 1, "third": 1, "trailing": 2,
	}))

	// Seed asymmetric file totals, then verify Captain uses them rather than round-robin.
	partitionDir := prepare(t)
	g.Expect(os.MkdirAll(filepath.Join(partitionDir, ".captain/pest"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(partitionDir, ".captain/pest/timings.yaml"), []byte(
		"tests/DescriptionsTest.php: 8s\ntests/ThirdTest.php: 2s\ntests/OtherTest.php: 1s\n"), 0o600)).To(Succeed())
	first := run(partitionDir, "partition", "pest", "tests/*Test.php", "--index", "0", "--total", "2")
	g.Expect(string(first)).To(ContainSubstring("tests/DescriptionsTest.php"))
	g.Expect(string(first)).NotTo(ContainSubstring("tests/ThirdTest.php"))
	second := run(partitionDir, "partition", "pest", "tests/*Test.php", "--index", "1", "--total", "2")
	g.Expect(string(second)).To(ContainSubstring("tests/ThirdTest.php tests/OtherTest.php"))
	g.Expect(string(second)).NotTo(ContainSubstring("No test file timings were matched"))

	seen := make(map[string]bool)
	for index := range 2 {
		run(partitionDir, "run", "pest", "--partition-index", strconv.Itoa(index), "--partition-total", "2",
			"--reporter", "rwx-v1-json=report.json")
		for _, test := range readResults(partitionDir).Tests {
			g.Expect(seen[test.Name]).To(BeFalse(), test.Name)
			seen[test.Name] = true
		}
		// Each shard normally has its own checkout and the same initial timing manifest.
		g.Expect(os.WriteFile(filepath.Join(partitionDir, ".captain/pest/timings.yaml"), []byte(
			"tests/DescriptionsTest.php: 8s\ntests/ThirdTest.php: 2s\ntests/OtherTest.php: 1s\n"), 0o600)).To(Succeed())
	}
	g.Expect(seen).To(HaveLen(10))
	empty := run(partitionDir, "run", "pest", "--partition-index", "3", "--partition-total", "4")
	g.Expect(string(empty)).To(ContainSubstring("partition"))
	data, err = os.ReadFile(filepath.Join(partitionDir, "attempts.json"))
	g.Expect(err).NotTo(HaveOccurred())
	var partitionAttempts map[string]int
	g.Expect(json.Unmarshal(data, &partitionAttempts)).To(Succeed())
	g.Expect(partitionAttempts).To(Equal(attempts))
}
