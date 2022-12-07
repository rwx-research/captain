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

var _ = Describe("RWXParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/rwx/v1.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RWXParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.RWXParser{}.Parse(strings.NewReader(`{"summary":`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like RWX", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.RWXParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.RWXParser{}.Parse(strings.NewReader(`{"tests": []}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.RWXParser{}.Parse(strings.NewReader(
				`{"$schema": "https://raw.githubusercontent.com/rwx-research/test-results-schema/main/v1.json"}`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
		})
	})
})
