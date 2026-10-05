//go:build phpunit

package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/gomega"
)

// Run with PHPUNIT_PHAR pointing to a PHPUnit 10.4+ PHAR and a built ../captain.
func TestPHPUnitPartition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	captain, err := filepath.Abs("../captain")
	g.Expect(err).NotTo(HaveOccurred())
	phar := os.Getenv("PHPUNIT_PHAR")
	g.Expect(phar).NotTo(BeEmpty(), "set PHPUNIT_PHAR to a PHPUnit PHAR")
	dir := t.TempDir()
	g.Expect(os.Mkdir(filepath.Join(dir, "tests"), 0o755)).To(Succeed())
	for _, name := range []string{"Slow", "Medium", "Fast"} {
		fixture := fmt.Sprintf(`<?php
use PHPUnit\Framework\TestCase;
final class %sTest extends TestCase {
    public function testPass(): void {
        file_put_contents('executed.txt', __CLASS__ . "\n", FILE_APPEND);
        usleep(1000);
        self::assertTrue(true);
    }
}
`, name)
		g.Expect(os.WriteFile(filepath.Join(dir, "tests", name+"Test.php"), []byte(fixture), 0o600)).To(Succeed())
	}
	// #nosec G703 -- dir is created by testing.T.TempDir, not user input.
	g.Expect(os.WriteFile(filepath.Join(dir, "captain.yaml"), []byte(fmt.Sprintf(`cloud:
  disabled: true
test-suites:
  phpunit:
    command: php %s --log-junit results.xml tests
    results:
      language: PHP
      framework: PHPUnit
      path: results.xml
    partition:
      globs: [tests/*Test.php]
      command: php %s --log-junit results.xml {{ testFiles }}
`, phar, phar)), 0o600)).To(Succeed())
	run := func(args ...string) string {
		// #nosec G204 -- execute the locally built Captain with test-owned arguments.
		cmd := exec.CommandContext(t.Context(), captain, append([]string{"--config-file", "captain.yaml"}, args...)...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred(), string(output))
		return string(output)
	}
	readResults := func() v1.TestResults {
		data, err := os.ReadFile(filepath.Join(dir, "report.json"))
		g.Expect(err).NotTo(HaveOccurred())
		var results v1.TestResults
		g.Expect(json.Unmarshal(data, &results)).To(Succeed())
		return results
	}
	run("run", "phpunit", "--update-stored-results", "--reporter", "rwx-v1-json=report.json")
	results := readResults()
	g.Expect(results.Framework).To(Equal(v1.PHPUnitFramework))
	g.Expect(results.Tests).To(HaveLen(3))
	expectedTimings := make(map[string]time.Duration)
	for _, test := range results.Tests {
		g.Expect(test.Location.File).To(HaveSuffix(".php"))
		g.Expect(test.Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
		expectedTimings[test.Location.File] += *test.Attempt.Duration
	}
	timingsPath := filepath.Join(dir, ".captain/phpunit/timings.yaml")
	data, err := os.ReadFile(timingsPath)
	g.Expect(err).NotTo(HaveOccurred())
	var timings map[string]time.Duration
	g.Expect(yaml.Unmarshal(data, &timings)).To(Succeed())
	g.Expect(timings).To(Equal(expectedTimings))
	g.Expect(os.WriteFile(timingsPath, []byte(
		"tests/SlowTest.php: 8s\ntests/MediumTest.php: 2s\ntests/FastTest.php: 1s\n"), 0o600)).To(Succeed())
	first := run("partition", "phpunit", "tests/*Test.php", "--index", "0", "--total", "2")
	g.Expect(first).To(ContainSubstring("tests/SlowTest.php"))
	g.Expect(first).NotTo(ContainSubstring("tests/MediumTest.php"))
	second := run("partition", "phpunit", "tests/*Test.php", "--index", "1", "--total", "2")
	g.Expect(second).To(ContainSubstring("tests/MediumTest.php tests/FastTest.php"))
	seen := make(map[string]bool)
	for index := range 2 {
		run("run", "phpunit", "--partition-index", strconv.Itoa(index), "--partition-total", "2",
			"--reporter", "rwx-v1-json=report.json")
		partition := readResults()
		g.Expect(partition.Tests).To(HaveLen(index + 1))
		for _, test := range partition.Tests {
			g.Expect(seen[test.Name]).To(BeFalse(), test.Name)
			seen[test.Name] = true
		}
	}
	g.Expect(seen).To(Equal(map[string]bool{
		"SlowTest::testPass": true, "MediumTest::testPass": true, "FastTest::testPass": true,
	}))
	before, err := os.ReadFile(filepath.Join(dir, "executed.txt"))
	g.Expect(err).NotTo(HaveOccurred())
	run("run", "phpunit", "--partition-index", "3", "--partition-total", "4")
	after, err := os.ReadFile(filepath.Join(dir, "executed.txt"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(after).To(Equal(before), "empty partition must not execute the suite")
}
