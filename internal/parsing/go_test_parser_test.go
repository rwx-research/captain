package parsing_test

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GoTestParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/go_test.jsonl")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.GoTestParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON with no remnants of Go Test JSON", func() {
			testResults, err := parsing.GoTestParser{}.Parse(strings.NewReader(`asdfasdfsdf`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Did not see any tests, so we cannot be sure it is go test output",
			))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like go test", func() {
			var testResults *v1.TestResults
			var err error

			// testResults, err = parsing.GoTestParser{}.Parse(strings.NewReader(`{}`))
			// Expect(err).To(HaveOccurred())
			// Expect(err.Error()).To(ContainSubstring("Test results do not look like go test"))
			// Expect(testResults).To(BeNil())

			testResults, err = parsing.GoTestParser{}.Parse(strings.NewReader(
				`
					{"Action":"run"}
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Test results do not look like go test"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.GoTestParser{}.Parse(strings.NewReader(
				`
					{"Package":"one"}
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Test results do not look like go test"))
			Expect(testResults).To(BeNil())
		})
	})
})
