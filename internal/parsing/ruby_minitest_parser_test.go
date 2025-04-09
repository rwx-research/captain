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

var _ = Describe("RubyMinitestParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/minitest.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyMinitestParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses empty files", func() {
			testResults, err := parsing.RubyMinitestParser{}.Parse(strings.NewReader(`<testsuites></testsuites>`))
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults.Tests).To(HaveLen(0))
		})

		It("errors on malformed XML", func() {
			testResults, err := parsing.RubyMinitestParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("errors on XML that doesn't look like minitest", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.RubyMinitestParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})
	})
})
