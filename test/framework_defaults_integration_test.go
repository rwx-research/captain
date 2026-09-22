//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI-only framework defaults integration", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	run := func(flags ...string) v1.TestResults {
		args := append([]string{"run", "default-suite", "--reporter", "rwx-v1-json=summary.json"}, flags...)
		cmd := captainCmd(captainArgs{args: args, env: map[string]string{"PATH": dir + ":" + os.Getenv("PATH")}})
		binary, err := filepath.Abs(cmd.Path)
		Expect(err).NotTo(HaveOccurred())
		cmd.Path = binary
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(output))
		Expect(string(output)).To(ContainSubstring("Captain"))
		data, err := os.ReadFile(filepath.Join(dir, "summary.json"))
		Expect(err).NotTo(HaveOccurred())
		var results v1.TestResults
		Expect(json.Unmarshal(data, &results)).To(Succeed())
		return results
	}

	DescribeTable("runs and retries RSpec with default templates", func(partition bool) {
		failed, err := os.ReadFile("fixtures/integration-tests/rspec-failed-not-quarantined.json")
		Expect(err).NotTo(HaveOccurred())
		passed, err := os.ReadFile("fixtures/integration-tests/rspec-passed.json")
		Expect(err).NotTo(HaveOccurred())
		passed = []byte(strings.ReplaceAll(string(passed), "is flaky", "is failing"))
		Expect(os.WriteFile(filepath.Join(dir, "failed.json"), failed, 0o600)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "passed.json"), passed, 0o600)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "bundle"), []byte(`#!/bin/sh
set -eu
printf '%s\n' "$*" >> invocations
if [ ! -f initial-run ]; then
  touch initial-run
  cp failed.json rspec.json
  exit 1
fi
cp passed.json rspec.json
`), 0o700)).To(Succeed())

		flags := []string{"--framework", "rspec", "--retries", "1"}
		initial := "exec rspec --format json --out rspec.json --format progress"
		if partition {
			Expect(os.Mkdir(filepath.Join(dir, "spec"), 0o700)).To(Succeed())
			for _, file := range []string{"a_spec.rb", "b_spec.rb", "c_spec.rb"} {
				Expect(os.WriteFile(filepath.Join(dir, "spec", file), nil, 0o600)).To(Succeed())
			}
			flags = append(flags, "--partition-index", "0", "--partition-total", "2", "--partition-round-robin")
			initial += " spec/a_spec.rb spec/c_spec.rb"
		}
		results := run(flags...)
		Expect(results.Framework).To(Equal(v1.RubyRSpecFramework))
		Expect(results.Tests).To(HaveLen(1))
		Expect(results.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
		Expect(results.Tests[0].PastAttempts).To(HaveLen(1))
		invocations, err := os.ReadFile(filepath.Join(dir, "invocations"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(invocations)).To(Equal(initial + "\n" +
			"exec rspec --format json --out rspec.json --format progress ./x.rb[1:1]\n"))
	}, Entry("unpartitioned", false), Entry("partitioned", true))

	It("parses the documented Cypress RWX output with the default path", func() {
		Expect(os.WriteFile(filepath.Join(dir, "npx"), []byte(`#!/bin/sh
set -eu
test "$*" = "cypress run"
mkdir -p tmp/rwx
cat > tmp/rwx/results.json <<'JSON'
{
  "$schema": "https://raw.githubusercontent.com/rwx-research/test-results-schema/main/v1.json",
  "framework": {"language": "JavaScript", "kind": "Cypress"},
  "tests": [{
    "name": "passes",
    "lineage": ["example", "passes"],
    "location": {"file": "cypress/e2e/example.cy.js"},
    "attempt": {"status": {"kind": "successful"}}
  }]
}
JSON
`), 0o700)).To(Succeed())
		results := run("--framework", "cypress")
		Expect(results.Framework).To(Equal(v1.JavaScriptCypressFramework))
		Expect(results.Tests).To(HaveLen(1))
		Expect(results.Tests[0].Name).To(Equal("passes"))
		Expect(results.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
	})
})
