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

		It("parses minitest-junit output", func() {
			fixture, err := os.Open("../../test/fixtures/minitest_junit.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyMinitestParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			Expect(testResults.Tests).To(HaveLen(4))

			byName := map[string]v1.Test{}
			for _, test := range testResults.Tests {
				byName[test.Name] = test
			}

			passed := byName["ExampleTest#test_passes"]
			Expect(passed.Location.File).To(Equal("test/example_test.rb"))
			Expect(*passed.Location.Line).To(Equal(4))
			Expect(passed.Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
			Expect(passed.Attempt.Meta["assertions"]).To(Equal(1))

			skipped := byName["ExampleTest#test_skips"]
			Expect(skipped.Location.File).To(Equal("test/example_test.rb"))
			Expect(*skipped.Location.Line).To(Equal(16))
			Expect(skipped.Attempt.Status.Kind).To(Equal(v1.TestStatusSkipped))

			failed := byName["ExampleTest#test_fails"]
			Expect(failed.Location.File).To(Equal("test/example_test.rb"))
			Expect(*failed.Location.Line).To(Equal(8))
			Expect(failed.Attempt.Status.Kind).To(Equal(v1.TestStatusFailed))
			Expect(*failed.Attempt.Status.Message).To(Equal("intentional failure"))
			Expect(failed.Attempt.Status.Backtrace).To(Equal([]string{"test/example_test.rb:9:in `test_fails'"}))

			errored := byName["ExampleTest#test_raises"]
			Expect(errored.Location.File).To(Equal("test/example_test.rb"))
			Expect(*errored.Location.Line).To(Equal(12))
			Expect(errored.Attempt.Status.Kind).To(Equal(v1.TestStatusFailed))
			Expect(*errored.Attempt.Status.Message).To(Equal("RuntimeError: uh oh"))
			Expect(errored.Attempt.Status.Backtrace).To(Equal([]string{"test/example_test.rb:13:in `test_raises'"}))

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
