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

var _ = Describe("GoGinkgoParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/ginkgo.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.GoGinkgoParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("parses other errors", func() {
			fixture, err := os.Open("../../test/fixtures/ginkgo_with_other_errors.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.GoGinkgoParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.GoGinkgoParser{}.Parse(strings.NewReader(`[{]`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors on JSON that doesn't look like Ginkgo", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.GoGinkgoParser{}.Parse(strings.NewReader(`[]`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Report is empty; unable to guarantee the results are Ginkgo"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.GoGinkgoParser{}.Parse(strings.NewReader(
				`
					[
						{},
						{"SuitePath": "foo"}
					]
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Report does not look like a Ginkgo report"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.GoGinkgoParser{}.Parse(strings.NewReader(
				`
					[
						{"SuitePath": "foo"},
						{}
					]
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Report does not look like a Ginkgo report"))
			Expect(testResults).To(BeNil())
		})
	})
})
