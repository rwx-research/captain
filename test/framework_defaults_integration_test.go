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

var _ = Describe("framework defaults integration", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	run := func(flags ...string) v1.TestResults {
		args := append([]string{"run", "default-suite", "--reporter", "rwx-v1-json=summary.json"}, flags...)
		cmd := captainCmd(captainArgs{args: args, env: map[string]string{
			"PATH": dir + ":" + os.Getenv("PATH"),
			"HOME": os.Getenv("HOME"),
		}})
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

	DescribeTable("runs and retries RSpec with default templates", func(partition, configured bool) {
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
		if configured {
			Expect(os.Mkdir(filepath.Join(dir, ".captain"), 0o700)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, ".captain", "config.yaml"), []byte(`test-suites:
  default-suite:
    results:
      framework: rspec
    retries:
      attempts: 1
`), 0o600)).To(Succeed())
			flags = nil
		}
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
	},
		Entry("unpartitioned", false, false),
		Entry("partitioned", true, false),
		Entry("configured unpartitioned", false, true),
		Entry("configured partitioned", true, true),
	)

	It("discovers Go packages and runs only the selected partition", func() {
		Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/suite\n\ngo 1.20\n"),
			0o600)).To(Succeed())
		for _, name := range []string{"a", "b", "c"} {
			Expect(os.Mkdir(filepath.Join(dir, name), 0o700)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, name, "example_test.go"),
				[]byte("package example\nimport \"testing\"\nfunc TestExample(t *testing.T) {}\n"), 0o600)).To(Succeed())
		}
		Expect(os.WriteFile(filepath.Join(dir, "gotestsum"), []byte(`#!/bin/sh
set -eu
printf '%s\n' "$*" > invocations
output=$3
shift 4
"$@" > "$output"
`), 0o700)).To(Succeed())
		results := run("--framework", "go test", "--partition-index", "0", "--partition-total", "2",
			"--partition-round-robin")
		Expect(results.Framework).To(Equal(v1.GoTestFramework))
		Expect(results.Tests).To(HaveLen(2))
		for _, test := range results.Tests {
			Expect(test.Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
		}
		invocations, err := os.ReadFile(filepath.Join(dir, "invocations"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(invocations)).To(Equal(
			"--raw-command --jsonfile go-test.json -- go test example.com/suite/a example.com/suite/c -json -count=1\n"))
	})

	It("runs Bun with its default command and results path", func() {
		Expect(os.WriteFile(filepath.Join(dir, "bun"), []byte(`#!/bin/sh
set -eu
printf '%s\n' "$@" > invocations
echo '<testsuites><testsuite name="example" tests="1"><testcase name="passes"/></testsuite></testsuites>' > bun.xml
`), 0o700)).To(Succeed())
		results := run("--framework", "bun")
		Expect(results.Tests).To(HaveLen(1))
		invocations, err := os.ReadFile(filepath.Join(dir, "invocations"))
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.Fields(string(invocations))).To(ConsistOf(
			"test", "--reporter=junit", "--reporter-outfile", "bun.xml"))
	})

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
